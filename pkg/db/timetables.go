package db

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type TimetablesRepo struct {
	db      orm.DB
	filters map[string][]Filter
	sort    map[string][]SortField
	join    map[string][]string
}

// NewTimetablesRepo returns new repository
func NewTimetablesRepo(db orm.DB) TimetablesRepo {
	return TimetablesRepo{
		db: db,
		filters: map[string][]Filter{
			Tables.Task.Name: {StatusFilter},
			Tables.User.Name: {StatusFilter},
		},
		sort: map[string][]SortField{
			Tables.Task.Name: {{Column: Columns.Task.CreatedAt, Direction: SortDesc}},
			Tables.User.Name: {{Column: Columns.User.ID, Direction: SortDesc}},
		},
		join: map[string][]string{
			Tables.Task.Name: {TableColumns, Columns.Task.UserTg},
			Tables.User.Name: {TableColumns},
		},
	}
}

// WithTransaction is a function that wraps TimetablesRepo with pg.Tx transaction.
func (tr TimetablesRepo) WithTransaction(tx *pg.Tx) TimetablesRepo {
	tr.db = tx
	return tr
}

// WithEnabledOnly is a function that adds "statusId"=1 as base filter.
func (tr TimetablesRepo) WithEnabledOnly() TimetablesRepo {
	f := make(map[string][]Filter, len(tr.filters))
	for i := range tr.filters {
		f[i] = make([]Filter, len(tr.filters[i]))
		copy(f[i], tr.filters[i])
		f[i] = append(f[i], StatusEnabledFilter)
	}
	tr.filters = f

	return tr
}

/*** Task ***/

// FullTask returns full joins with all columns
func (tr TimetablesRepo) FullTask() OpFunc {
	return WithColumns(tr.join[Tables.Task.Name]...)
}

// DefaultTaskSort returns default sort.
func (tr TimetablesRepo) DefaultTaskSort() OpFunc {
	return WithSort(tr.sort[Tables.Task.Name]...)
}

// TaskByID is a function that returns Task by ID(s) or nil.
func (tr TimetablesRepo) TaskByID(ctx context.Context, id int, ops ...OpFunc) (*Task, error) {
	return tr.OneTask(ctx, &TaskSearch{ID: &id}, ops...)
}

// OneTask is a function that returns one Task by filters. It could return pg.ErrMultiRows.
func (tr TimetablesRepo) OneTask(ctx context.Context, search *TaskSearch, ops ...OpFunc) (*Task, error) {
	obj := &Task{}
	err := buildQuery(ctx, tr.db, obj, search, tr.filters[Tables.Task.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// TasksByFilters returns Task list.
func (tr TimetablesRepo) TasksByFilters(ctx context.Context, search *TaskSearch, pager Pager, ops ...OpFunc) (tasks []Task, err error) {
	err = buildQuery(ctx, tr.db, &tasks, search, tr.filters[Tables.Task.Name], pager, ops...).Select()
	return
}

// CountTasks returns count
func (tr TimetablesRepo) CountTasks(ctx context.Context, search *TaskSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, tr.db, &Task{}, search, tr.filters[Tables.Task.Name], PagerOne, ops...).Count()
}

// AddTask adds Task to DB.
func (tr TimetablesRepo) AddTask(ctx context.Context, task *Task, ops ...OpFunc) (*Task, error) {
	q := tr.db.ModelContext(ctx, task)
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Task.CreatedAt)
	}
	applyOps(q, ops...)
	_, err := q.Insert()

	return task, err
}

// UpdateTask updates Task in DB.
func (tr TimetablesRepo) UpdateTask(ctx context.Context, task *Task, ops ...OpFunc) (bool, error) {
	q := tr.db.ModelContext(ctx, task).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Task.ID, Columns.Task.CreatedAt)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteTask set statusId to deleted in DB.
func (tr TimetablesRepo) DeleteTask(ctx context.Context, id int) (deleted bool, err error) {
	task := &Task{ID: id, StatusID: StatusDeleted}

	return tr.UpdateTask(ctx, task, WithColumns(Columns.Task.StatusID))
}

/*** User ***/

// FullUser returns full joins with all columns
func (tr TimetablesRepo) FullUser() OpFunc {
	return WithColumns(tr.join[Tables.User.Name]...)
}

// DefaultUserSort returns default sort.
func (tr TimetablesRepo) DefaultUserSort() OpFunc {
	return WithSort(tr.sort[Tables.User.Name]...)
}

// UserByID is a function that returns User by ID(s) or nil.
func (tr TimetablesRepo) UserByID(ctx context.Context, id int, ops ...OpFunc) (*User, error) {
	return tr.OneUser(ctx, &UserSearch{ID: &id}, ops...)
}

// OneUser is a function that returns one User by filters. It could return pg.ErrMultiRows.
func (tr TimetablesRepo) OneUser(ctx context.Context, search *UserSearch, ops ...OpFunc) (*User, error) {
	obj := &User{}
	err := buildQuery(ctx, tr.db, obj, search, tr.filters[Tables.User.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// UsersByFilters returns User list.
func (tr TimetablesRepo) UsersByFilters(ctx context.Context, search *UserSearch, pager Pager, ops ...OpFunc) (users []User, err error) {
	err = buildQuery(ctx, tr.db, &users, search, tr.filters[Tables.User.Name], pager, ops...).Select()
	return
}

// CountUsers returns count
func (tr TimetablesRepo) CountUsers(ctx context.Context, search *UserSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, tr.db, &User{}, search, tr.filters[Tables.User.Name], PagerOne, ops...).Count()
}

// AddUser adds User to DB.
func (tr TimetablesRepo) AddUser(ctx context.Context, user *User, ops ...OpFunc) (*User, error) {
	q := tr.db.ModelContext(ctx, user)
	applyOps(q, ops...)
	_, err := q.Insert()

	return user, err
}

// UpdateUser updates User in DB.
func (tr TimetablesRepo) UpdateUser(ctx context.Context, user *User, ops ...OpFunc) (bool, error) {
	q := tr.db.ModelContext(ctx, user).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.User.ID)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteUser set statusId to deleted in DB.
func (tr TimetablesRepo) DeleteUser(ctx context.Context, id int) (deleted bool, err error) {
	user := &User{ID: id, StatusID: StatusDeleted}

	return tr.UpdateUser(ctx, user, WithColumns(Columns.User.StatusID))
}
