package rbac

import (
	"fmt"
	"github.com/fluxxu/util"
	"github.com/lann/squirrel"
	"time"
)

func NewItem(t ItemType, name string) *item {
	return &item{
		Type: t,
		Name: name,
	}
}

func LoadItem(name string) (*item, error) {
	i := &item{}
	err := dbx.Get(i, "SELECT * FROM rbacitem WHERE name = ?", name)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *item) validate() (*util.ValidationContext, error) {
	v := util.NewValidationContext()

	if i.Name == "" {
		v.AddError("name", "Name is required")
	} else {
		var n int
		err := dbx.Get(&n, "SELECT COUNT(*) FROM rbacitem WHERE name = ?", i.Name)
		if err != nil {
			return nil, err
		}
		if n != 0 {
			v.AddError("name", "Duplicate Name")
		}
	}
	return v, nil
}

func (i *item) LoadChildren() error {
	if i.Children != nil {
		return nil
	}
	children, err := LoadChildren(i.Name)
	if err != nil {
		return err
	}
	i.Children = children
	return nil
}

func LoadChildren(name string) ([]item, error) {
	var children []item
	err := dbx.Select(&children, `
		SELECT rbacitem.* FROM rbacitemchild JOIN rbacitem ON rbacitemchild.parent = ? AND rbacitemchild.child = rbacitem.name`,
		name)
	if err != nil {
		return nil, err
	}
	return children, nil
}

//TODO support rule
func (i *item) Insert() error {
	v, err := i.validate()
	if err != nil {
		return nil
	}

	if v.HasError() {
		return v.ToError()
	}

	now := time.Now()
	q := squirrel.Insert("rbacitem")
	q = q.Columns("name", "type", "description", "rule_name", "data", "created_at")
	q = q.Values(i.Name, i.Type, i.Description, nil, nil, now)
	_, err = q.RunWith(dbx.DB).Exec()
	if err != nil {
		return err
	}
	return nil
}

func (i *item) Delete() error {
	if i.Name == "" {
		return fmt.Errorf("no name")
	}

	if _, err := dbx.Exec("DELETE FROM rbacitem WHERE name = ? LIMIT 1", i.Name); err != nil {
		return err
	}

	return nil
}
