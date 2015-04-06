package rbac

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zenazn/goji"
	"testing"
)

func TestCreateItem(t *testing.T) {
	item := NewItem(TypeOperation, "test")
	item.Description = "desc"
	require.NoError(t, item.Insert())

	item, err := LoadItem("test")
	require.NoError(t, err)

	assert.Equal(t, "test", item.Name)
	assert.Equal(t, "desc", item.Description)
}

func TestAddChild(t *testing.T) {
	role := NewItem(TypeRole, "testrole")
	require.NoError(t, role.Insert())

	task := NewItem(TypeTask, "testtask")
	require.NoError(t, task.Insert())

	task = NewItem(TypeTask, "testtask2")
	require.NoError(t, task.Insert())

	op := NewItem(TypeOperation, "testop")
	require.NoError(t, op.Insert())

	op = NewItem(TypeOperation, "testop2")
	require.NoError(t, op.Insert())

	require.NoError(t, AddItemChild("testtask", "testop"))
	require.NoError(t, AddItemChild("testtask", "testop2"))

	require.NoError(t, AddItemChild("testrole", "testtask"))

	items, err := LoadChildren("testrole")
	require.NoError(t, err)
	require.Equal(t, 1, len(items))
	require.Equal(t, "testtask", items[0].Name)

	items, err = LoadChildren("testtask")
	require.NoError(t, err)
	require.Equal(t, 2, len(items))
	require.Equal(t, "testop", items[0].Name)
	require.Equal(t, "testop2", items[1].Name)

	items, err = LoadChildren("testtask1")
	require.NoError(t, err)
	require.Equal(t, 0, len(items))
}

func TestCheckAccess(t *testing.T) {
	role := NewItem(TypeRole, "checkrole")
	require.NoError(t, role.Insert())

	task := NewItem(TypeTask, "checktask")
	require.NoError(t, task.Insert())

	task = NewItem(TypeTask, "checktask2")
	require.NoError(t, task.Insert())

	op := NewItem(TypeOperation, "checkop")
	require.NoError(t, op.Insert())

	op = NewItem(TypeOperation, "checkop2")
	require.NoError(t, op.Insert())

	require.NoError(t, AddItemChild("checktask", "checkop"))
	require.NoError(t, AddItemChild("checktask", "checkop2"))

	require.NoError(t, AddItemChild("checkrole", "checktask"))

	check := func(item string) bool {
		ok, err := CheckAccess(item, "u")
		require.NoError(t, err)
		return ok
	}

	assert.Equal(t, false, check("checkrole"))

	require.NoError(t, Assign("checkrole", "u"))

	assert.Equal(t, true, check("checkrole"))
	assert.Equal(t, true, check("checktask"))
	assert.Equal(t, false, check("checktask2"))
	assert.Equal(t, true, check("checkop"))
	assert.Equal(t, true, check("checkop2"))
}

func TestBatchQuery(t *testing.T) {
	role := NewItem(TypeRole, "batchrole1")
	require.NoError(t, role.Insert())

	role = NewItem(TypeRole, "batchrole2")
	require.NoError(t, role.Insert())

	role = NewItem(TypeRole, "batchrole3")
	require.NoError(t, role.Insert())

	require.NoError(t, NewItem(TypeTask, "task").Insert())
	require.NoError(t, Assign("task", "u1"))
	require.NoError(t, Assign("batchrole1", "u1"))

	require.NoError(t, Assign("batchrole2", "u2"))
	require.NoError(t, Assign("batchrole3", "u2"))

	r, err := BatchQuery(TypeRole, []string{"u1", "u2", "u3"})
	require.NoError(t, err)

	assert.Len(t, r[0], 1)
	assert.Contains(t, r[0], "batchrole1")

	assert.Len(t, r[1], 2)
	assert.Contains(t, r[1], "batchrole2")
	assert.Contains(t, r[1], "batchrole3")

	assert.Len(t, r[2], 0)
}

func TestQuery(t *testing.T) {
	require.NoError(t, NewItem(TypeRole, "queryrole1").Insert())
	require.NoError(t, NewItem(TypeTask, "querytask1").Insert())
	require.NoError(t, NewItem(TypeTask, "querytask2").Insert())
	require.NoError(t, Assign("queryrole1", "queryu"))
	require.NoError(t, Assign("querytask1", "queryu"))
	require.NoError(t, Assign("querytask2", "queryu"))
	items, err := Query(TypeAny, "queryu")
	require.NoError(t, err)
	require.Equal(t, 3, len(items))
	assert.Contains(t, items, "queryrole1")
	assert.Contains(t, items, "querytask1")
	assert.Contains(t, items, "querytask2")

	items, err = Query(TypeTask, "queryu")
	require.NoError(t, err)
	require.Equal(t, 2, len(items))
	assert.Contains(t, items, "querytask1")
	assert.Contains(t, items, "querytask2")

	items, err = Query(TypeRole, "queryu")
	require.NoError(t, err)
	require.Equal(t, 1, len(items))
	assert.Contains(t, items, "queryrole1")
}

func TestSync(t *testing.T) {
	check := func(tp ItemType, args ...string) {
		items, err := Query(tp, "syncu")
		require.NoError(t, err)
		assert.Equal(t, len(args), len(items))
		for _, i := range args {
			assert.Contains(t, items, i)
		}
	}

	role := NewItem(TypeRole, "sync1")
	require.NoError(t, role.Insert())

	role = NewItem(TypeRole, "sync2")
	require.NoError(t, role.Insert())

	role = NewItem(TypeRole, "sync3")
	require.NoError(t, role.Insert())

	require.NoError(t, Assign("sync1", "syncu"))
	check(TypeAny, "sync1")

	require.NoError(t, Sync("syncu", []string{}))
	check(TypeAny)

	require.NoError(t, Sync("syncu", []string{"sync1", "sync3"}))
	check(TypeAny, "sync1", "sync3")

	require.NoError(t, Sync("syncu", []string{"sync2", "sync3"}))
	check(TypeAny, "sync2", "sync3")
}

func init() {
	dbx := sqlx.MustConnect("mysql", "root@/test?parseTime=true")
	Configure(&Opts{
		Dbx:     dbx,
		Mux:     goji.DefaultMux,
		MuxBase: "/",
	})

	dbx.MustExec("DELETE FROM rbacitem")
}
