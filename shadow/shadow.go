package shadow

import (
	"encoding/json"
	"time"

	v1 "github.com/baetyl/baetyl-go/spec/v1"
	bh "github.com/timshannon/bolthold"
	bolt "go.etcd.io/bbolt"
)

// Shadow shadow
type Shadow struct {
	bk    []byte
	id    []byte
	store *bh.Store
}

// NewShadow create a new shadow
func NewShadow(namespace, name string, store *bh.Store) (*Shadow, error) {
	m := &v1.Shadow{
		Name:              name,
		Namespace:         namespace,
		CreationTimestamp: time.Now(),
		Report:            v1.Report{},
		Desire:            v1.Desire{},
	}
	s := &Shadow{
		bk:    []byte("baetyl-edge-shadow"),
		id:    []byte(name + "." + namespace),
		store: store,
	}
	err := s.insert(m)
	if err != nil && err != bh.ErrKeyExists {
		return nil, err
	}
	return s, nil
}

// Get returns shadow model
func (s *Shadow) Get() (m *v1.Shadow, err error) {
	err = s.store.Bolt().View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bk)
		prev := b.Get(s.id)
		if len(prev) == 0 {
			return bh.ErrNotFound
		}
		m = &v1.Shadow{}
		return json.Unmarshal(prev, m)
	})
	return
}

// Desire update shadow desired data, then return the delta of desired and reported data
func (s *Shadow) Desire(desired v1.Desire) (delta v1.Desire, err error) {
	err = s.store.Bolt().Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bk)
		prev := b.Get(s.id)
		if len(prev) == 0 {
			return bh.ErrNotFound
		}
		m := &v1.Shadow{}
		err := json.Unmarshal(prev, m)
		if err != nil {
			return err
		}
		if m.Desire == nil {
			m.Desire = desired
		} else {
			err = m.Desire.Merge(desired)
			if err != nil {
				return err
			}
		}
		curr, err := json.Marshal(m)
		if err != nil {
			return err
		}
		err = b.Put(s.id, curr)
		if err != nil {
			return err
		}
		delta, err = m.Desire.Diff(m.Report)
		return err
	})
	return
}

// Report update shadow reported data, then return the delta of desired and reported data
func (s *Shadow) Report(reported v1.Report) (delta v1.Desire, err error) {
	err = s.store.Bolt().Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bk)
		prev := b.Get(s.id)
		if len(prev) == 0 {
			return bh.ErrNotFound
		}
		m := &v1.Shadow{}
		err := json.Unmarshal(prev, m)
		if err != nil {
			return err
		}
		if m.Report == nil {
			m.Report = reported
		} else {
			err = m.Report.Merge(reported)
			if err != nil {
				return err
			}
		}
		curr, err := json.Marshal(m)
		if err != nil {
			return err
		}
		err = b.Put(s.id, curr)
		if err != nil {
			return err
		}
		delta, err = m.Desire.Diff(m.Report)
		return err
	})
	return
}

// Get insert the whole shadow data
func (s *Shadow) insert(m *v1.Shadow) error {
	return s.store.Bolt().Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(s.bk)
		if err != nil {
			return err
		}
		data := b.Get(s.id)
		if len(data) != 0 {
			return bh.ErrKeyExists
		}
		data, err = json.Marshal(m)
		if err != nil {
			return err
		}
		return b.Put(s.id, data)
	})
}
