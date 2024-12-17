package orm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BaseRepo interface {
	Save(ctx context.Context, record DBModel) error

	BatchSave(ctx context.Context, value interface{}) error

	Upsert(ctx context.Context, updateColumns []string, conflictColumns []string, record DBModel) error

	BatchUpsert(ctx context.Context, updateColumns []string, conflictColumns []string, value interface{}) error

	Update(ctx context.Context, record DBModel) error

	UpdateByFields(ctx context.Context, filters *FieldFilter, record DBModel, columns ...string) error

	GetOneByID(ctx context.Context, id string, record DBModel) error

	GetOneByFields(ctx context.Context, filters *FieldFilter, record DBModel, columns ...string) error

	QueryByFields(ctx context.Context, filters *FieldFilter, table DBModel, result interface{}, limit int, columns ...string) error
}

type DBModel interface {
	Database(sharding bool) string

	TableName() string
}
type mysqlBaseRepo struct {
	data DbProvider
}

type DbProvider interface {
	GetDB(database string) (*gorm.DB, bool)
	Sharding() bool
}

func (m *mysqlBaseRepo) Save(ctx context.Context, record DBModel) error {
	db, err := GetGorm(ctx, record, m.data)
	if err != nil {
		return err
	}
	if err = db.Create(record).Error; err != nil {
		return fmt.Errorf("[database]:%s  [table]:%s ,Save err[%v]", record.Database(m.data.Sharding()), record.TableName(), err)
	}
	return nil

}

func (m *mysqlBaseRepo) BatchSave(ctx context.Context, value interface{}) error {
	dbModel, batchSize, err := fetchDbModel(value)
	if err != nil {
		return err
	}
	db, err := GetGorm(ctx, dbModel, m.data)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	if err = db.CreateInBatches(value, batchSize).Error; err != nil {
		return fmt.Errorf("[database]:%s  [table]:%s ,BatchSave err[%v]", dbModel.Database(m.data.Sharding()), dbModel.TableName(), err)
	}
	return nil
}

func (m *mysqlBaseRepo) Upsert(ctx context.Context,
	updateColumns []string,
	conflictColumns []string,
	record DBModel) error {
	db, err := GetGorm(ctx, record, m.data)
	if err != nil {
		return err
	}
	var columns []clause.Column
	for _, column := range conflictColumns {
		columns = append(columns, clause.Column{Name: column})
	}
	if err = db.Clauses(clause.OnConflict{
		Columns:   columns,                                 // key colume
		DoUpdates: clause.AssignmentColumns(updateColumns), // column needed to be updated
	}).Create(record).Error; err != nil {
		return fmt.Errorf("[database]:%s  [table]:%s ,Upsert err[%v]", record.Database(m.data.Sharding()), record.TableName(), err)
	}
	return nil
}

func (m *mysqlBaseRepo) BatchUpsert(ctx context.Context,
	updateColumns []string,
	conflictColumns []string,
	value interface{}) error {

	var columns []clause.Column
	for _, column := range conflictColumns {
		columns = append(columns, clause.Column{Name: column})
	}
	dbModel, batchSize, err := fetchDbModel(value)
	if err != nil {
		return err
	}
	db, err := GetGorm(ctx, dbModel, m.data)
	if err != nil {
		return err
	}
	if err = db.Clauses(clause.OnConflict{
		Columns:   columns,                                 // key colume
		DoUpdates: clause.AssignmentColumns(updateColumns), // column needed to be updated
	}).CreateInBatches(value, batchSize).Error; err != nil {
		return fmt.Errorf("[database]:%s  [table]:%s ,BatchUpsert err[%v]", dbModel.Database(m.data.Sharding()), dbModel.TableName(), err)
	}
	return nil

}

func (m *mysqlBaseRepo) Update(ctx context.Context, record DBModel) error {
	db, err := GetGorm(ctx, record, m.data)
	if err != nil {
		return err
	}
	if err = db.Updates(record).Error; err != nil {
		return fmt.Errorf("[database]:%s  [table]:%s ,Update err[%v]", record.Database(m.data.Sharding()), record.TableName(), err)
	}
	return nil
}

func (m *mysqlBaseRepo) GetOneByID(ctx context.Context, id string, record DBModel) error {
	db, err := GetGorm(ctx, record, m.data)
	if err != nil {
		return err
	}
	if err = db.Where("id = ?", id).First(record).Error; err != nil {
		return fmt.Errorf("[database]:%s  [table]:%s,[id]: %s ,GetOneByID err[%v]", record.Database(m.data.Sharding()), record.TableName(), id, err)
	}
	return nil
}

func (m *mysqlBaseRepo) GetOneByFields(ctx context.Context, filters *FieldFilter,
	record DBModel, columns ...string) error {
	db, err := GetGorm(ctx, record, m.data)
	if err != nil {
		return err
	}
	if len(columns) > 0 {
		db = db.Select(columns)
	}
	db, err = filters.FillCondition(db, record)
	if err != nil {
		return err
	}
	if err = db.First(record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("[database]:%s  [table]:%s ,GetOneByFields err[%v]", record.Database(m.data.Sharding()), record.TableName(), err)
	}
	return nil
}

func (m *mysqlBaseRepo) QueryByFields(ctx context.Context,
	filters *FieldFilter, table DBModel, result interface{}, limit int, columns ...string) error {
	db, err := GetGorm(ctx, table, m.data)
	if err != nil {
		return err
	}
	if len(columns) > 0 {
		db = db.Select(columns)
	}
	db, err = filters.FillCondition(db, table)
	if err != nil {
		return err
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err = db.Find(result).Error; err != nil {
		return fmt.Errorf("[database]:%s  [table]:%s ,QueryByFields err[%v]", table.Database(m.data.Sharding()), table.TableName(), err)
	}
	return nil
}
func (m *mysqlBaseRepo) UpdateByFields(ctx context.Context, filters *FieldFilter, table DBModel, columns ...string) error {
	db, err := GetGorm(ctx, table, m.data)
	if err != nil {
		return err
	}
	db, err = filters.FillCondition(db, table)
	if err != nil {
		return err
	}
	if len(columns) > 0 {
		db = db.Select(columns)
	}
	if err = db.Updates(table).Error; err != nil {
		return err
	}
	return nil
}
func NewBaseRepo(data DbProvider) BaseRepo {
	return &mysqlBaseRepo{
		data: data,
	}
}

func GetGorm(ctx context.Context, record DBModel, data DbProvider) (*gorm.DB, error) {
	if db, ok := GetTxContext(ctx); ok {
		return db, nil
	}
	if db, ok := data.GetDB(record.Database(data.Sharding())); ok {
		return db, nil
	}
	return nil, fmt.Errorf("not found database,record[%v]", record)
}

func fetchDbModel(value interface{}) (DBModel, int, error) {
	reflectValue := reflect.Indirect(reflect.ValueOf(value))

	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return nil, 0, fmt.Errorf("value is need slice %v", value)
	}
	val := reflectValue.Index(0).Interface()
	dbModel, ok := val.(DBModel)
	if !ok {
		return nil, 0, fmt.Errorf("value is need DBModel %v", value)
	}
	return dbModel, reflectValue.Len(), nil
}

type FieldFilter struct {
	conditions []FilterCondition
}

func NewFieldFilter() *FieldFilter {
	return &FieldFilter{}
}
func (ff *FieldFilter) FillCondition(db *gorm.DB, table DBModel) (*gorm.DB, error) {
	var err error
	for _, condition := range ff.conditions {
		db, err = condition.buildCondition(db)
		if err != nil {
			return nil, fmt.Errorf("[database]:%s  [table]:%s ,QueryByFields err[%v]", table.Database(false), table.TableName(), err)
		}
	}
	return db, nil
}

func (ff *FieldFilter) Add(key string, val interface{}) {
	ff.conditions = append(ff.conditions, &Eq{
		column: key,
		value:  val,
		op:     EqOp,
	})
}
func (ff *FieldFilter) AddNeq(key string, val interface{}) {
	ff.conditions = append(ff.conditions, &Neq{Eq{
		column: key,
		value:  val,
		op:     NeqOp,
	}})
}
func (ff *FieldFilter) AddGte(key string, val interface{}) {
	ff.conditions = append(ff.conditions, &Gte{Eq{
		column: key,
		value:  val,
		op:     GteOp,
	}})
}
func (ff *FieldFilter) AddLte(key string, val interface{}) {
	ff.conditions = append(ff.conditions, &Lte{Eq{
		column: key,
		value:  val,
		op:     LteOp,
	}})
}
func (ff *FieldFilter) AddNotIn(key string, val interface{}) {
	ff.conditions = append(ff.conditions, &NotIN{IN{Eq: Eq{
		column: key,
		value:  val,
		op:     NotInOp,
	},
	},
	})
}
func (ff *FieldFilter) AddIn(key string, val interface{}) {
	ff.conditions = append(ff.conditions, &IN{Eq: Eq{
		column: key,
		value:  val,
		op:     InOp,
	},
	})
}
func (ff *FieldFilter) AddLike(key string, val interface{}) {
	ff.conditions = append(ff.conditions, &Like{Eq: Eq{
		column: key,
		value:  val,
		op:     LikeOp,
	},
	})
}

type OP string

const (
	EqOp    OP = "="
	NeqOp   OP = "<>"
	GteOp   OP = ">="
	LteOp   OP = "<="
	NotInOp OP = "NOT IN"
	InOp    OP = "IN"
	LikeOp  OP = "LIKE"
)

type FilterCondition interface {
	buildCondition(db *gorm.DB) (*gorm.DB, error)
}
type Eq struct {
	column string
	value  interface{}
	op     OP
}

func (eq *Eq) buildCondition(db *gorm.DB) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%v %s ?", eq.column, eq.op), eq.value), nil
}

type Neq struct {
	Eq
}

type Gte struct {
	Eq
}
type Lte struct {
	Eq
}

type Like struct {
	Eq
}

func (like *Like) buildCondition(db *gorm.DB) (*gorm.DB, error) {
	return db.Where(fmt.Sprintf("%v %s %%?%%", like.column, like.op), like.value), nil
}

type IN struct {
	Eq
}
type NotIN struct {
	IN
}

func (in *IN) buildCondition(db *gorm.DB) (*gorm.DB, error) {
	reflectValue := reflect.Indirect(reflect.ValueOf(in.value))

	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return nil, fmt.Errorf("value is need slice %v", in.value)
	}
	if reflectValue.Len() == 0 {
		return db, nil
	}
	var params []interface{}
	placeholder := ""
	for i := 0; i < reflectValue.Len(); i++ {
		params = append(params, reflectValue.Index(i).Interface())
		if reflectValue.Len() == i+1 {
			placeholder += "?"
		} else {
			placeholder += "?,"
		}
	}
	return db.Where(fmt.Sprintf("%s %s (%s)", in.column, in.op, placeholder), params...), nil
}

type QueryCondition interface {
}

func BuildFieldFilter(condition QueryCondition) *FieldFilter {
	val := reflect.Indirect(reflect.ValueOf(condition))
	typeV := reflect.TypeOf(condition)
	if typeV.Kind() == reflect.Pointer {
		typeV = typeV.Elem()
	}
	fieldFilter := NewFieldFilter()
	for index := 0; index < typeV.NumField(); index++ {
		field := typeV.Field(index)
		tag := field.Tag.Get(QueryCondTagName)
		if len(tag) <= 0 {
			continue
		}
		if val.FieldByName(field.Name).IsZero() {
			continue
		}
		tagMap := ParseTagSetting(tag, Sep)
		queryCond := &QueryCondTag{
			Value:      val.FieldByName(field.Name).Interface(),
			TagSetting: tagMap,
			Column:     CamelCaseToUnderscore(field.Name),
		}
		queryCond.AddFieldFilter(fieldFilter)
	}
	return fieldFilter
}

const (
	Sep              = ";"
	OPTagName        = "op"
	AliasTagName     = "alias"
	QueryCondTagName = "queryCond"
)

type QueryCondTag struct {
	Op         OP
	Alias      string
	Value      interface{}
	Column     string
	TagSetting map[string]string
}

func (tag *QueryCondTag) GetOP() OP {
	op, ok := tag.TagSetting[strings.ToUpper(OPTagName)]
	if !ok || len(op) == 0 {
		return EqOp
	}
	op = strings.ToUpper(op)
	return OP(op)
}
func (tag *QueryCondTag) GetColumn() string {
	alias, ok := tag.TagSetting[strings.ToUpper(AliasTagName)]
	if ok && len(alias) > 0 {
		return alias
	}
	return tag.Column
}
func (tag *QueryCondTag) AddFieldFilter(ff *FieldFilter) {
	switch tag.GetOP() {
	case LikeOp:
		ff.AddLike(tag.GetColumn(), tag.Value)
		break
	case EqOp:
		ff.Add(tag.GetColumn(), tag.Value)
		break
	case InOp:
		ff.AddIn(tag.GetColumn(), tag.Value)
		break
	case NotInOp:
		ff.AddNotIn(tag.GetColumn(), tag.Value)
		break
	case LteOp:
		ff.AddLte(tag.GetColumn(), tag.Value)
		break
	case GteOp:
		ff.AddGte(tag.GetColumn(), tag.Value)
		break
	case NeqOp:
		ff.AddNeq(tag.GetColumn(), tag.Value)
		break
	}
}

func ParseTagSetting(str string, sep string) map[string]string {
	settings := map[string]string{}
	names := strings.Split(str, sep)

	for i := 0; i < len(names); i++ {
		j := i
		if len(names[j]) > 0 {
			for {
				if names[j][len(names[j])-1] == '\\' {
					i++
					names[j] = names[j][0:len(names[j])-1] + sep + names[i]
					names[i] = ""
				} else {
					break
				}
			}
		}

		values := strings.Split(names[j], ":")
		k := strings.TrimSpace(strings.ToUpper(values[0]))

		if len(values) >= 2 {
			settings[k] = strings.Join(values[1:], ":")
		} else if k != "" {
			settings[k] = k
		}
	}

	return settings
}
func CamelCaseToUnderscore(s string) string {
	var output []rune
	for i, r := range s {
		if i == 0 {
			output = append(output, unicode.ToLower(r))
		} else {
			if unicode.IsUpper(r) {
				output = append(output, '_')
			}

			output = append(output, unicode.ToLower(r))
		}
	}
	return string(output)
}
