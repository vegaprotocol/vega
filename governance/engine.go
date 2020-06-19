package governance

import (
	"sync"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	ErrProposalNotFound    = errors.New("proposal not found")
	ErrProposalIsDuplicate = errors.New("proposal with given ID already exists")
	// Validation errors

	ErrProposalCloseTimeInvalid   = errors.New("proposal closes too soon or too late")
	ErrProposalEnactTimeInvalid   = errors.New("proposal enactment times too soon or late")
	ErrProposalInsufficientTokens = errors.New("proposal requires more tokens than party has")

	ErrVoterInsufficientTokens = errors.New("vote requires more tokens than party has")
	ErrVotePeriodExpired       = errors.New("proposal voting has been closed")

	ErrAssetProposalReferenceDuplicate = errors.New("duplicate asset proposal for reference")
)

// Accounts ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/accounts_mock.go -package mocks code.vegaprotocol.io/vega/governance Accounts
type Accounts interface {
	GetPartyTokenAccount(id string) (*types.Account, error)
	GetTotalTokens() uint64
}

// Buffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/proposal_buffer_mock.go -package mocks code.vegaprotocol.io/vega/governance Buffer
type Buffer interface {
	Add(types.Proposal)
	Flush()
}

// VoteBuf...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vote_buffer_mock.go -package mocks code.vegaprotocol.io/vega/governance VoteBuf
type VoteBuf interface {
	Add(types.Vote)
	Flush()
}

// ValidatorTopology...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_topology_mock.go -package mocks code.vegaprotocol.io/vega/governance ValidatorTopology
type ValidatorTopology interface {
	Exists([]byte) bool
	Len() int
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/wallet_mock.go -package mocks code.vegaprotocol.io/vega/governance Wallet
type Wallet interface {
	Get(chain nodewallet.Blockchain) (nodewallet.Wallet, bool)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/commander_mock.go -package mocks code.vegaprotocol.io/vega/governance Commander
type Commander interface {
	Command(key nodewallet.Wallet, cmd blockchain.Command, payload proto.Message) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/governance Assets
type Assets interface {
	NewAsset(ref string, assetSrc *types.AssetSource) (string, error)
	Get(assetID string) (assets.Asset, error)
}

type network struct {
	minClose, maxClose, minEnact, maxEnact int64
	participation                          uint64
}

type Engine struct {
	Config
	accs         Accounts
	buf          Buffer
	vbuf         VoteBuf
	log          *logging.Logger
	mu           sync.Mutex
	currentTime  time.Time
	proposals    map[string]*proposalVote
	proposalRefs map[string]*proposalVote
	net          network
	isValidator  bool

	nodeProposalValidation *NodeValidation
}

type proposalVote struct {
	*types.Proposal
	yes map[string]*types.Vote
	no  map[string]*types.Vote
}

func NewEngine(log *logging.Logger, cfg Config, accs Accounts, buf Buffer, vbuf VoteBuf, top ValidatorTopology, wallet Wallet, cmd Commander, assets Assets, now time.Time, isValidator bool) (*Engine, error) {
	log = log.Named(namedLogger)
	// ensure params are set
	cfg.initParams()

	nodeValidation, err := NewNodeValidation(log, top, wallet, cmd, assets, now, isValidator)
	if err != nil {
		return nil, err
	}

	return &Engine{
		Config:       cfg,
		accs:         accs,
		buf:          buf,
		vbuf:         vbuf,
		log:          log,
		currentTime:  now,
		proposals:    map[string]*proposalVote{},
		proposalRefs: map[string]*proposalVote{},
		net: network{
			minClose:      cfg.params.DefaultMinClose,
			maxClose:      cfg.params.DefaultMaxClose,
			minEnact:      cfg.params.DefaultMinEnact,
			maxEnact:      cfg.params.DefaultMaxEnact,
			participation: cfg.params.DefaultMinParticipation,
		},
		nodeProposalValidation: nodeValidation,
	}, nil
}

// ReloadConf updates the internal configuration of the collateral engine
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.mu.Lock()
	cfg.params = e.Config.params
	e.Config = cfg
	e.mu.Unlock()
}

// OnChainUpdate - update curtime, expire proposals
func (e *Engine) OnChainTimeUpdate(t time.Time) []*types.Proposal {
	e.currentTime = t
	now := t.Unix()
	expired := []*proposalVote{}
	for k, p := range e.proposals {
		// only if we're passed the valid unitl, letting in the last of the votes
		if p.Terms.ClosingTimestamp < now {
			expired = append(expired, p)
			delete(e.proposals, k)
			delete(e.proposalRefs, p.Reference) // remove from ref map, Foo
		}
	}

	// then get all proposal accepted through node validation, and start their vote time.
	for _, p := range e.nodeProposalValidation.OnChainTimeUpdate(t) {
		e.log.Info("proposal has been validated by nodes, starting now",
			logging.String("proposal-id", p.ID))
		e.startProposal(p) // can't fail, and proposal has been validated at an ulterior time
	}

	// flush here for now
	e.buf.Flush()
	e.vbuf.Flush()
	return e.processProposals(expired)
}

func (e *Engine) AddProposal(p types.Proposal) error {
	// @TODO -> we probably should keep proposals in memory here
	if cp, ok := e.proposals[p.ID]; ok && cp.State == p.State {
		return ErrProposalIsDuplicate
	}
	if _, ok := e.proposalRefs[p.Reference]; ok {
		return ErrProposalIsDuplicate
	}
	var err error
	if err = e.validateProposal(p); err != nil {
		p.State = types.Proposal_REJECTED
	}
	if p.State != types.Proposal_OPEN {
		delete(e.proposals, p.ID)
		delete(e.proposalRefs, p.Reference)
	} else {
		// now if it's a 2 steps proposal, start the node votes
		if e.isTwoStepsProposal(&p) {
			err = e.startTwoStepsProposal(&p)
		} else {
			e.startProposal(&p)
		}
	}
	e.buf.Add(p)
	return err
}

func (e *Engine) startProposal(p *types.Proposal) {
	pv := proposalVote{
		Proposal: p,
		yes:      map[string]*types.Vote{},
		no:       map[string]*types.Vote{},
	}
	e.proposals[p.ID] = &pv
	e.proposalRefs[p.Reference] = &pv
}

func (e *Engine) startTwoStepsProposal(p *types.Proposal) error {
	return e.nodeProposalValidation.Start(p)
}

func (e *Engine) isTwoStepsProposal(p *types.Proposal) bool {
	return e.nodeProposalValidation.IsNodeValidationRequired(p)
}

func (e *Engine) validateProposal(p types.Proposal) error {
	tok, err := e.accs.GetPartyTokenAccount(p.PartyID)
	if err != nil {
		return err
	}
	if tok.Balance < 1 {
		return ErrProposalInsufficientTokens
	}

	minClose, maxClose := e.currentTime.Add(time.Duration(e.net.minClose)*time.Second),
		e.currentTime.Add(time.Duration(e.net.maxClose)*time.Second)
	if p.Terms.ClosingTimestamp < minClose.Unix() || p.Terms.ClosingTimestamp > maxClose.Unix() {
		return ErrProposalCloseTimeInvalid
	}

	minEnact, maxEnact := p.Terms.ClosingTimestamp, p.Terms.ClosingTimestamp+e.net.maxEnact
	if p.Terms.EnactmentTimestamp < minEnact || p.Terms.EnactmentTimestamp > maxEnact {
		return ErrProposalEnactTimeInvalid
	}

	return nil
}

func (e *Engine) AddNodeVote(v *types.NodeVote) error {
	return e.nodeProposalValidation.AddNodeVote(v)
}

func (e *Engine) AddVote(v types.Vote) error {
	p, err := e.validateVote(v)
	if err != nil {
		return err
	}
	// we only want to count the last vote, so add to yes/no map, delete from the other
	// if the party hasn't cast a vote yet, the delete is just a noop
	if v.Value == types.Vote_YES {
		delete(p.no, v.PartyID)
		p.yes[v.PartyID] = &v
	} else {
		delete(p.yes, v.PartyID)
		p.no[v.PartyID] = &v
	}
	e.vbuf.Add(v)
	return nil
}

func (e *Engine) validateVote(v types.Vote) (*proposalVote, error) {
	tacc, err := e.accs.GetPartyTokenAccount(v.PartyID)
	if err != nil {
		return nil, err
	}
	if tacc.Balance == 0 {
		return nil, ErrVoterInsufficientTokens
	}
	p, ok := e.proposals[v.ProposalID]
	if !ok {
		return nil, ErrProposalNotFound
	}
	if p.Terms.ClosingTimestamp < e.currentTime.Unix() {
		return nil, ErrVotePeriodExpired
	}
	return p, nil
}

func (e *Engine) processProposals(proposals []*proposalVote) []*types.Proposal {
	// we're calculating based off of percentages
	allTokens := e.accs.GetTotalTokens()
	// 1 % of tokens represented here
	tokPercent := float64(allTokens) / 100.0
	accepted := make([]*types.Proposal, 0, len(proposals))
	buf := map[string]*types.Account{}
	var err error
	for _, pw := range proposals {
		p := pw.Proposal
		var totalYES uint64
		for _, v := range pw.yes {
			tok, ok := buf[v.PartyID]
			if !ok {
				tok, err = e.accs.GetPartyTokenAccount(v.PartyID)
				if err != nil {
					e.log.Error(
						"Failed to get account for party",
						logging.String("party-id", v.PartyID),
						logging.Error(err),
					)
					break
				}
			}
			totalYES += tok.Balance
		}
		p.State = types.Proposal_DECLINED
		// participation stake used as a percentage required to approve the proposal
		reqTokens := tokPercent * float64(p.Terms.MinParticipationStake)
		if reqTokens <= float64(totalYES) {
			p.State = types.Proposal_PASSED
			accepted = append(accepted, p)
		}
		e.buf.Add(*p)
	}
	return accepted
}
