package rbac

import (
	"database/sql"
	"fmt"
	"github.com/fluxxu/util"
	"time"
)

const MaxCheckRecursionLevel int = 3

type ItemType int

const (
	TypeRole ItemType = iota
	TypeTask
	TypeOperation
)

type item struct {
	Name        string         `db:"name" json:"name"`
	Type        ItemType       `db:"type" json:"type"`
	Description string         `db:"description" json:"description"`
	RuleName    sql.NullString `db:"rule_name" json:"role_name"`
	Data        interface{}    `db:"data" json:"-"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   util.NullTime  `db:"updated_at" json:"updated_at"`

	Children []item
}

type rule struct {
	Name      string        `db:"name" json:"name"`
	Data      interface{}   `db:"data" json:"-"`
	CreatedAt time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt util.NullTime `db:"updated_at" json:"updated_at"`
}

type assignment struct {
	ItemName  string    `db:"item_name" json:"item_name"`
	UserId    string    `db:"user_id" json:"user_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func Assign(item string, userId string) error {
	_, err := dbx.Exec("INSERT IGNORE INTO rbacassignment(item_name, user_id, created_at) VALUES(?, ?, ?)", item, userId, time.Now())
	return err
}

func Revoke(item string, userId string) error {
	_, err := dbx.Exec("DELETE FROM rbacassignment WHERE item_name = ? AND user_id = ? LIMIT 1", item, userId)
	return err
}

func AddItemChild(item string, child string) error {
	var n int
	err := dbx.Get(&n, "SELECT COUNT(*) FROM rbacitem WHERE name = ? OR name = ?", item, child)
	if err != nil {
		return err
	}

	if n != 2 {
		return fmt.Errorf("item or child not found")
	}

	if item == child {
		return fmt.Errorf("self == child")
	}

	_, err = dbx.Exec("INSERT IGNORE INTO rbacitemchild(parent, child) VALUES(?, ?)", item, child)
	if err != nil {
		return err
	}
	return nil
}

func RemoveItemChild(item string, child string) error {
	_, err := dbx.Exec("DELETE FROM rbacitemchild WHERE parent = ? AND child = ?", item, child)
	return err
}

func CheckAccess(item string, userId string) (bool, error) {
	var assignments []assignment
	if err := dbx.Select(&assignments, "SELECT * FROM rbacassignment WHERE user_id = ?", userId); err != nil {
		return false, err
	}

	var checkChildrenRecursively func(int, string, string) (bool, error)
	checkChildrenRecursively = func(level int, item string, parent string) (bool, error) {
		if level > MaxCheckRecursionLevel {
			return false, fmt.Errorf("MaxCheckRecursionLevel reached, item: %s", item)
		}

		children, err := LoadChildren(parent)
		if err != nil {
			return false, err
		}

		for _, child := range children {
			if child.Name == item {
				return true, nil
			}

			ok, err := checkChildrenRecursively(level+1, item, child.Name)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}

	for _, a := range assignments {
		if a.ItemName == item {
			//TODO support rule
			return true, nil
		}

		ok, err := checkChildrenRecursively(0, item, a.ItemName)
		if err != nil {
			return false, err
		}

		if ok {
			return true, nil
		}
	}
	return false, nil
}
