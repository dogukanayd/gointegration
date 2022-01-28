package gointegration

import "testing"

func TestConfigs_NewUnit(t *testing.T) {
	cfg := Configs{
		DatabaseName: "slaDB",
		SQLFilePath:  "/pkg/mysql/mysql-dump",
	}
	db, tearDown, err := cfg.NewUnit(t)

	if err != nil {
		t.Error(err)
	}

	defer tearDown()

	_, err = db.Exec("INSERT INTO slaDB.business_slas (product, feature, partner_name, phase, unit_id, unit_name, started_at) VALUES ('architect', 'test', 'inshoppingcart', 'started', 88, 'super_builder_id', 1637844045)")

	if err != nil {
		t.Error(err)
	}
}
