package db

import (
	"context"
	"errors"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
)

type CommonRepo struct {
	db      orm.DB
	filters map[string][]Filter
	sort    map[string][]SortField
	join    map[string][]string
}

// NewCommonRepo returns new repository
func NewCommonRepo(db orm.DB) CommonRepo {
	return CommonRepo{
		db: db,
		filters: map[string][]Filter{
			Tables.User.Name: {StatusFilter},
		},
		sort: map[string][]SortField{
			Tables.User.Name:  {{Column: Columns.User.CreatedAt, Direction: SortDesc}},
			Tables.Place.Name: {{Column: Columns.Place.ID, Direction: SortDesc}},
		},
		join: map[string][]string{
			Tables.User.Name:  {TableColumns},
			Tables.Place.Name: {TableColumns, Columns.Place.User},
		},
	}
}

// WithTransaction is a function that wraps CommonRepo with pg.Tx transaction.
func (cr CommonRepo) WithTransaction(tx *pg.Tx) CommonRepo {
	cr.db = tx
	return cr
}

// WithEnabledOnly is a function that adds "statusId"=1 as base filter.
func (cr CommonRepo) WithEnabledOnly() CommonRepo {
	f := make(map[string][]Filter, len(cr.filters))
	for i := range cr.filters {
		f[i] = make([]Filter, len(cr.filters[i]))
		copy(f[i], cr.filters[i])
		f[i] = append(f[i], StatusEnabledFilter)
	}
	cr.filters = f

	return cr
}

/*** User ***/

// FullUser returns full joins with all columns
func (cr CommonRepo) FullUser() OpFunc {
	return WithColumns(cr.join[Tables.User.Name]...)
}

// DefaultUserSort returns default sort.
func (cr CommonRepo) DefaultUserSort() OpFunc {
	return WithSort(cr.sort[Tables.User.Name]...)
}

// UserByID is a function that returns User by ID(s) or nil.
func (cr CommonRepo) UserByID(ctx context.Context, id int, ops ...OpFunc) (*User, error) {
	return cr.OneUser(ctx, &UserSearch{ID: &id}, ops...)
}

// OneUser is a function that returns one User by filters. It could return pg.ErrMultiRows.
func (cr CommonRepo) OneUser(ctx context.Context, search *UserSearch, ops ...OpFunc) (*User, error) {
	obj := &User{}
	err := buildQuery(ctx, cr.db, obj, search, cr.filters[Tables.User.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// UsersByFilters returns User list.
func (cr CommonRepo) UsersByFilters(ctx context.Context, search *UserSearch, pager Pager, ops ...OpFunc) (users []User, err error) {
	err = buildQuery(ctx, cr.db, &users, search, cr.filters[Tables.User.Name], pager, ops...).Select()
	return
}

// CountUsers returns count
func (cr CommonRepo) CountUsers(ctx context.Context, search *UserSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, cr.db, &User{}, search, cr.filters[Tables.User.Name], PagerOne, ops...).Count()
}

// AddUser adds User to DB.
func (cr CommonRepo) AddUser(ctx context.Context, user *User, ops ...OpFunc) (*User, error) {
	q := cr.db.ModelContext(ctx, user)
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.User.CreatedAt)
	}
	applyOps(q, ops...)
	_, err := q.Insert()

	return user, err
}

// UpdateUser updates User in DB.
func (cr CommonRepo) UpdateUser(ctx context.Context, user *User, ops ...OpFunc) (bool, error) {
	q := cr.db.ModelContext(ctx, user).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.User.ID, Columns.User.CreatedAt)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeleteUser set statusId to deleted in DB.
func (cr CommonRepo) DeleteUser(ctx context.Context, id int) (deleted bool, err error) {
	user := &User{ID: id, StatusID: StatusDeleted}

	return cr.UpdateUser(ctx, user, WithColumns(Columns.User.StatusID))
}

/*** Place ***/

// FullPlace returns full joins with all columns
func (cr CommonRepo) FullPlace() OpFunc {
	return WithColumns(cr.join[Tables.Place.Name]...)
}

// DefaultPlaceSort returns default sort.
func (cr CommonRepo) DefaultPlaceSort() OpFunc {
	return WithSort(cr.sort[Tables.Place.Name]...)
}

// PlaceByID is a function that returns Place by ID(s) or nil.
func (cr CommonRepo) PlaceByID(ctx context.Context, id int, ops ...OpFunc) (*Place, error) {
	return cr.OnePlace(ctx, &PlaceSearch{ID: &id}, ops...)
}

// OnePlace is a function that returns one Place by filters. It could return pg.ErrMultiRows.
func (cr CommonRepo) OnePlace(ctx context.Context, search *PlaceSearch, ops ...OpFunc) (*Place, error) {
	obj := &Place{}
	err := buildQuery(ctx, cr.db, obj, search, cr.filters[Tables.Place.Name], PagerTwo, ops...).Select()

	if errors.Is(err, pg.ErrMultiRows) {
		return nil, err
	} else if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	}

	return obj, err
}

// PlacesByFilters returns Place list.
func (cr CommonRepo) PlacesByFilters(ctx context.Context, search *PlaceSearch, pager Pager, ops ...OpFunc) (places []Place, err error) {
	err = buildQuery(ctx, cr.db, &places, search, cr.filters[Tables.Place.Name], pager, ops...).Select()
	return
}

// CountPlaces returns count
func (cr CommonRepo) CountPlaces(ctx context.Context, search *PlaceSearch, ops ...OpFunc) (int, error) {
	return buildQuery(ctx, cr.db, &Place{}, search, cr.filters[Tables.Place.Name], PagerOne, ops...).Count()
}

// AddPlace adds Place to DB.
func (cr CommonRepo) AddPlace(ctx context.Context, place *Place, ops ...OpFunc) (*Place, error) {
	q := cr.db.ModelContext(ctx, place)
	applyOps(q, ops...)
	_, err := q.Insert()

	return place, err
}

// UpdatePlace updates Place in DB.
func (cr CommonRepo) UpdatePlace(ctx context.Context, place *Place, ops ...OpFunc) (bool, error) {
	q := cr.db.ModelContext(ctx, place).WherePK()
	if len(ops) == 0 {
		q = q.ExcludeColumn(Columns.Place.ID)
	}
	applyOps(q, ops...)
	res, err := q.Update()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}

// DeletePlace deletes Place from DB.
func (cr CommonRepo) DeletePlace(ctx context.Context, id int) (deleted bool, err error) {
	place := &Place{ID: id}

	res, err := cr.db.ModelContext(ctx, place).WherePK().Delete()
	if err != nil {
		return false, err
	}

	return res.RowsAffected() > 0, err
}
