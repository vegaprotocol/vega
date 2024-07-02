-- +goose Up

WITH updated_pnl AS (
    SELECT DISTINCT ON (pc.party_id) pc.party_id AS pid, pc.market_id AS mid, pc.loss - ph.loss AS correct_loss, pc.loss_socialisation_amount - ph.loss_socialisation_amount AS correct_loss_soc,
		pc.realised_pnl - ph.realised_pnl AS correct_pnl,
		pc.pending_realised_pnl - ph.pending_realised_pnl AS correct_ppnl,
		pc.adjustment - ph.adjustment AS correct_adj
	FROM positions_current AS pc
	JOIN positions AS ph
		ON pc.party_id = ph.party_id
		AND pc.market_id = ph.market_id
	WHERE pc.party_id IN ('\x947a700141e3d175304ee176d0beecf9ee9f462e09330e33c386952caf21f679', '\x15a8f372e255c6fa596a0b3acd62bc3be63b65188c23d33fc350f38ef52902e3', '\xaa1ce33b0b31a2e0f0a947ba83f64fa4a7e5d977fffb82c278c3b33fb0498113', '\x6527ffdd223ef2b4695ad90d832adc5493e9b8e25ad3185e67d873767f1f275e')
		AND ph.vega_time >= '2024-06-08 19:38:49.89053+00'
		AND pc.market_id = '\xe63a37edae8b74599d976f5dedbf3316af82579447f7a08ae0495a021fd44d13'
	ORDER BY pc.party_id, ph.vega_time ASC
)
UPDATE positions_current
SET loss = updated_pnl.correct_loss,
    loss_socialisation_amount = updated_pnl.correct_loss_soc,
    realised_pnl = updated_pnl.correct_pnl,
    pending_realised_pnl = updated_pnl.correct_ppnl,
	adjustment = updated_pnl.correct_adj
FROM updated_pnl
WHERE party_id = updated_pnl.pid AND market_id = updated_pnl.mid;

-- +goose Down
-- nothing
