package rbac

import (
	"database/sql"
	"fmt"
	"github.com/fluxxu/util"
	"github.com/lann/squirrel"
	"time"
)

const MaxCheckRecursionLevel int = 3

type ItemType int

const (
	TypeAny ItemType = iota
	TypeRole
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

func Query(t ItemType, userId string) ([]string, error) {
	rv := []string{}
	q := squirrel.Select("rbacitem.name").From("rbacassignment").Join("rbacitem ON rbacitem.name = rbacassignment.item_name")
	if t != TypeAny {
		q = q.Where("rbacitem.type = ?", t)
	}
	q = q.Where("rbacassignment.user_id = ?", userId)
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	if err = dbx.Select(&rv, sql, args...); err != nil {
		return nil, err
	}
	return rv, nil
}

func BatchQuery(t ItemType, userIdSet []string) ([][]string, error) {
	count := len(userIdSet)
	if count == 0 {
		return [][]string{}, nil
	}

	iMap := make(map[string]int)
	for index, user := range userIdSet {
		iMap[user] = index
	}
	rv := make([][]string, count)

	q := squirrel.Select("rbacassignment.user_id, rbacitem.name").From("rbacassignment").Join("rbacitem ON rbacitem.name = rbacassignment.item_name")
	if t != TypeAny {
		q = q.Where("rbacitem.type = ?", t)
	}
	q = q.Where(squirrel.Eq{"rbacassignment.user_id": userIdSet})
	rows, err := q.RunWith(dbx.DB).Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var user string
		var role string
		if err = rows.Scan(&user, &role); err != nil {
			return nil, err
		}
		index := iMap[user]
		rv[index] = append(rv[index], role)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return rv, nil
}

func Sync(userId string, items []string) error {
	existing := []string{}
	if err := dbx.Select(&existing, "SELECT item_name FROM rbacassignment WHERE user_id = ?", userId); err != nil {
		return err
	}

	removeList := []string{}
	insertList := []string{}

	for _, existingItem := range existing {
		if util.IndexOfString(items, existingItem) == -1 {
			removeList = append(removeList, existingItem)
		}
	}

	for _, item := range items {
		if util.IndexOfString(existing, item) == -1 {
			insertList = append(insertList, item)
		}
	}

	if len(removeList) > 0 || len(insertList) > 0 {
		tx, err := dbx.Begin()
		if err != nil {
			return err
		}

		rollback := func(err error) error {
			if txerr := tx.Rollback(); txerr != nil {
				return fmt.Errorf("%s; %s", err.Error(), txerr.Error())
			}
			return err
		}

		if len(removeList) > 0 {
			q := squirrel.Delete("rbacassignment").Where("user_id = ?", userId).Where(squirrel.Eq{"item_name": removeList})
			sql, args, err := q.ToSql()
			if err != nil {
				return rollback(err)
			}

			if _, err = tx.Exec(sql, args...); err != nil {
				return rollback(err)
			}
		}

		if len(insertList) > 0 {
			q := squirrel.Insert("rbacassignment").Columns("item_name", "user_id", "created_at")
			for _, item := range insertList {
				q = q.Values(item, userId, time.Now())
			}
			sql, args, err := q.ToSql()
			if err != nil {
				return rollback(err)
			}

			if _, err = tx.Exec(sql, args...); err != nil {
				return rollback(err)
			}
		}

		return tx.Commit()
	}
	return nil
}
