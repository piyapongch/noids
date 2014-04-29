package server

import (
	"database/sql"
	"log"
	"time"
)

type dbSaver struct {
	DB *sql.DB
}

const dbSchema = `CREATE TABLE noids (
name STRING,
template STRING,
closed BOOLEAN,
lastmint STRING
);`

// Create a PoolSaver which will serialize noid pools as
// records in a SQL database
func NewDbFileSaver(db *sql.DB) PoolSaver {
	// see if db works and create table if necessary
	return &dbSaver{DB: db}
}

func (d *dbSaver) SavePool(name string, pi PoolInfo) error {
	log.Println("Save (db)", name)
	lastmintText, err := pi.LastMint.MarshalText()
	result, err := d.DB.Exec("UPDATE 'noids' SET 'template' = ?, 'closed' = ?, 'lastmint' = ? WHERE 'name' = ?", pi.Template, pi.Closed, lastmintText, name)
	if err != nil {
		return err
	}
	nrows, err := result.RowsAffected()
	if err != nil {
		// driver does not support row count
		// see if the record is in the database in the first place
		// TODO(dbrower)
		return err
	}
	switch {
	case nrows == 0:
		_, err = d.DB.Exec("INSERT INTO noids VALUES (?, ?, ?, ?)", name, pi.Template, pi.Closed, lastmintText)
	case nrows == 1:
	default:
		log.Printf("There is more than one row in the database for pool '%s'", name)
		// TODO(dbrower): make error constant for this
		err = nil
	}
	return err
}

func (d *dbSaver) LoadAllPools() ([]PoolInfo, error) {
	var pis []PoolInfo

	rows, err := d.DB.Query("SELECT name, template, closed, lastmint FROM noids")
	if err != nil {
		return pis, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			name, template, lastmint sql.NullString
			closed                   sql.NullBool
			lm                       time.Time
		)
		err := rows.Scan(&name, &template, &closed, &lastmint)
		if err != nil {
			return pis, err
		}
		err = (&lm).UnmarshalText([]byte(lastmint.String))
		if err != nil {
			return pis, err
		}
		pi := PoolInfo{
			Name:     name.String,
			Template: template.String,
			Closed:   closed.Bool,
			LastMint: lm,
		}
		pis = append(pis, pi)
	}
	if err := rows.Err(); err != nil {
		return pis, err
	}
	return pis, nil
}
