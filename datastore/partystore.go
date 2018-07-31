package datastore

type memPartyStore struct {
	store *MemStore
}

// NewPartyStore initialises a new PartyStore backed by a MemStore.
func NewPartyStore(ms *MemStore) PartyStore {
	return &memPartyStore{store: ms}
}

func (m *memPartyStore) Post(party string) error {
	return nil
}

func (m *memPartyStore) Put(party string) error {
	return nil
}

func (m *memPartyStore) Delete(party string) error {
	return nil
}


func (m *memPartyStore) GetAllParties() (parties []string, err error) {
	for party, _ := range m.store.parties {
		parties = append(parties, party)
	}
	return parties, nil
}