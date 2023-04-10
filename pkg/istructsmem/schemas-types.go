/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"

	"github.com/untillpro/dynobuffers"
	"github.com/untillpro/voedger/pkg/istructs"
	payloads "github.com/untillpro/voedger/pkg/itokens-payloads"
)

// SchemasCacheType application schemas
//   - interfaces:
//     — istructs.ISchemas
type SchemasCacheType struct {
	appCfg  *AppConfigType
	schemas map[istructs.QName]*SchemaType
}

func newSchemaCache(appCfg *AppConfigType) SchemasCacheType {
	cache := SchemasCacheType{
		appCfg:  appCfg,
		schemas: make(map[istructs.QName]*SchemaType),
	}
	return cache
}

func fullKeyName(viewName istructs.QName) istructs.QName {
	const fullKeyFmt = "%s_FullKey"
	return istructs.NewQName(viewName.Pkg(), fmt.Sprintf(fullKeyFmt, viewName.Entity()))
}

// buildViewKeySchema for specified view key schema constructs new full key schema, contained fields from partition key and clustering columns key shemas
func (cache *SchemasCacheType) buildViewKeySchema(schema *SchemaType) error {
	if err := schema.Validate(true); err != nil {
		return err
	}

	partCont := schema.containers[istructs.SystemContainer_ViewPartitionKey]
	partSchema := cache.schemaByName(partCont.schema)
	clustCont := schema.containers[istructs.SystemContainer_ViewClusteringCols]
	clustSchema := cache.schemaByName(clustCont.schema)

	keySchema := cache.Add(fullKeyName(schema.name), istructs.SchemaKind_ViewRecord_ClusteringColumns)
	for _, name := range partSchema.fieldsOrder {
		fld := partSchema.fields[name]
		keySchema.AddField(name, fld.kind, false)
	}
	for _, name := range clustSchema.fieldsOrder {
		fld := clustSchema.fields[name]
		keySchema.AddField(name, fld.kind, false)
	}

	return keySchema.Validate(false)
}

// buildViewKeySchemas for all viewRecord schemas build full key schemas
func (cache *SchemasCacheType) buildViewKeySchemas() error {
	viewSchemas := make(map[istructs.QName]*SchemaType)
	for _, sch := range cache.schemas {
		if sch.kind == istructs.SchemaKind_ViewRecord {
			viewSchemas[sch.name] = sch
		}
	}

	for _, sch := range viewSchemas {
		if err := cache.buildViewKeySchema(sch); err != nil {
			return err
		}
	}

	return nil
}

// Add adds new schema specified name and kind
func (cache *SchemasCacheType) Add(name istructs.QName, kind istructs.SchemaKindType) (schema *SchemaType) {
	schema, ok := cache.schemas[name]
	if ok {
		panic(fmt.Errorf("schema name «%s» already used: %w", name, ErrNameUniqueViolation))
	}
	schema = newSchema(cache, name, kind)
	cache.schemas[name] = schema
	return schema
}

// Add adds new schema specified name and kind
func (cache *SchemasCacheType) AddView(name istructs.QName) *ViewSchemaType {
	v := newViewSchema(cache, name)
	return &v
}

// Schema returns schema by name
func (cache *SchemasCacheType) Schema(schema istructs.QName) istructs.ISchema {
	s := cache.schemaByName(schema)
	if s == nil {
		return NullSchema
	}
	return s
}

// Schemas enumerates all application resources
func (cache *SchemasCacheType) Schemas(enum func(istructs.QName)) {
	for n := range cache.schemas {
		enum(n)
	}
}

// validEvent validate specified event. Must be called _after_ build() method
func (cache *SchemasCacheType) validEvent(ev *eventType) (err error) {
	// if ev.name == istructs.NullQName — unnecessary check, must be checked in ev.build()

	if err := cache.validEventObjects(ev); err != nil {
		return err
	}

	if err := cache.validEventCUDs(ev); err != nil {
		return err
	}

	return nil
}

// validEventObjects validate event parts: object and unlogged object
func (cache *SchemasCacheType) validEventObjects(ev *eventType) (err error) {
	arg, argUnl, err := ev.argumentNames()
	if err != nil {
		return validateError(ECode_InvalidSchemaName, err)
	}

	if ev.argObject.QName() != arg {
		return validateErrorf(ECode_InvalidSchemaName, "event command argument «%v» uses wrong schema «%v», expected «%v»: %w", ev.name, ev.argObject.QName(), arg, ErrWrongSchema)
	}

	if arg != istructs.NullQName {
		// #!17185: must be ODoc or Object only
		schema := cache.schemaByName(arg)

		// if schema == nil — unnecessary check, must be checked in ev.build()

		if (schema.kind != istructs.SchemaKind_ODoc) && (schema.kind != istructs.SchemaKind_Object) {
			return validateErrorf(ECode_InvalidSchemaKind, "event command argument «%v» schema can not to be «%s», expected («%s» or «%s»): %w", arg, shemaKindToStr[schema.kind], sk_ODoc, sk_Object, ErrWrongSchema)
		}
		if err := cache.validObject(&ev.argObject); err != nil {
			return err
		}
	}

	if ev.argUnlObj.QName() != argUnl {
		return validateErrorf(ECode_InvalidSchemaName, "event command unlogged argument «%v» uses wrong schema «%v», expected «%v»: %w", ev.name, ev.argUnlObj.QName(), argUnl, ErrWrongSchema)
	}

	if ev.argUnlObj.QName() != istructs.NullQName {
		if err := cache.validObject(&ev.argUnlObj); err != nil {
			return err
		}
	}

	return nil
}

// validEventCUDs validate event CUD parts: argument CUDs and result CUDs
func (cache *SchemasCacheType) validEventCUDs(ev *eventType) (err error) {
	if ev.cud.empty() {
		if ev.name == istructs.QNameCommandCUD {
			return validateErrorf(ECode_EEmptyCUDs, "event «%v» must have not empty CUDs: %w", ev.name, ErrCUDsMissed)
		}
		return nil
	}

	return cache.validCUD(&ev.cud, ev.sync)
}

// validObject validates specified document or object
func (cache *SchemasCacheType) validObject(obj *elementType) (err error) {
	if obj.QName() == istructs.NullQName {
		return validateErrorf(ECode_EmptySchemaName, "element «%s» has empty schema name: %w", obj.Container(), ErrNameMissed)
	}

	schema := cache.schemaByName(obj.QName())

	switch schema.kind {
	case istructs.SchemaKind_GDoc, istructs.SchemaKind_CDoc, istructs.SchemaKind_ODoc, istructs.SchemaKind_WDoc:
		{
			err = schema.validDocument(obj)
			if err != nil {
				return err
			}
		}
	case istructs.SchemaKind_Object:
		{
			err = schema.validObject(obj)
			if err != nil {
				return err
			}
		}
	default:
		{
			return validateErrorf(ECode_InvalidSchemaKind, "object refers to invalid schema «%v» kind «%s»: %w", schema.name, shemaKindToStr[schema.kind], ErrUnexpectedShemaKind)
		}
	}

	return nil
}

// validCUD validates specified CUD
func (cache *SchemasCacheType) validCUD(cud *cudType, allowStorageIDsInCreate bool) (err error) {
	for _, newRec := range cud.creates {
		if err = cache.validRecord(newRec, !allowStorageIDsInCreate); err != nil {
			return err
		}
	}

	if err = cache.validCUDIDsUnique(cud); err != nil {
		return err
	}

	if err = cache.validCUDRefRawIDs(cud); err != nil {
		return err
	}

	for _, updRec := range cud.updates {
		if err = cache.validRecord(&updRec.result, false); err != nil {
			return err
		}
	}

	return nil
}

// validCUDIDsUnique validates IDs in CUD for unique
func (cache *SchemasCacheType) validCUDIDsUnique(cud *cudType) (err error) {
	const errRecIDViolatedWrap = "record ID «%d» is used repeatedly: %w"

	ids := make(map[istructs.RecordID]bool)

	for _, rec := range cud.creates {
		id := rec.ID()
		if _, exists := ids[id]; exists {
			return validateErrorf(ECode_InvalidRecordID, errRecIDViolatedWrap, id, ErrRecordIDUniqueViolation)
		}
		ids[id] = true
	}
	for _, rec := range cud.updates {
		id := rec.changes.ID()
		if _, exists := ids[id]; exists {
			return validateErrorf(ECode_InvalidRecordID, errRecIDViolatedWrap, id, ErrRecordIDUniqueViolation)
		}
		ids[id] = true
	}

	return nil
}

// validCUDRefRawIDs validates references to raw IDs in specified CUD
func (cache *SchemasCacheType) validCUDRefRawIDs(cud *cudType) (err error) {

	rawIDs := make(map[istructs.RecordID]bool)

	for _, rec := range cud.creates {
		id := rec.ID()
		if id.IsRaw() {
			rawIDs[id] = true
		}
	}

	checkRefs := func(rec *recordType) (err error) {
		rec.RecordIDs(false,
			func(name string, id istructs.RecordID) {
				if err != nil {
					return
				}
				if id.IsRaw() && !rawIDs[id] {
					err = validateErrorf(ECode_InvalidRefRecordID, "record «%s» field «%s» refers to unknown raw ID «%d»: %w", rec.Container(), name, id, ErrorRecordIDNotFound)
				}
			})
		return err
	}

	for _, rec := range cud.creates {
		if err := checkRefs(rec); err != nil {
			return err
		}
	}

	for _, rec := range cud.updates {
		if err := checkRefs(&rec.changes); err != nil {
			return err
		}
	}

	return nil
}

// validKey validates specified view key. If partialClust specified then clustering columns row may be partially filled
func (cache *SchemasCacheType) validKey(key *keyType, partialClust bool) (err error) {
	partSchema := key.partKeySchema()
	if key.partRow.QName() != partSchema {
		return validateErrorf(ECode_InvalidSchemaName, "wrong view partition key schema «%v», for view «%v» expected «%v»: %w", key.partRow.QName(), key.viewName, partSchema, ErrWrongSchema)
	}
	for n := range key.partRow.schema.fields {
		if !key.partRow.hasValue(n) {
			return validateErrorf(ECode_EmptyData, "view «%v» partition key «%v» field «%s» is empty: %w", key.viewName, partSchema, n, ErrFieldIsEmpty)
		}
	}
	// if err = key.partRow.schema.validRow(&key.partRow); err != nil {…} — unnecessary check, already checked in for-loop above

	clustSchema := key.clustColsSchema()
	if key.clustRow.QName() != clustSchema {
		return validateErrorf(ECode_InvalidSchemaName, "wrong view clustering columns schema «%v», for view «%v» expected «%v»: %w", key.clustRow.QName(), key.viewName, clustSchema, ErrWrongSchema)
	}
	if !partialClust {
		for n := range key.clustRow.schema.fields {
			if !key.clustRow.hasValue(n) {
				return validateErrorf(ECode_EmptyData, "view «%v» clustering columns «%v» field «%s» is empty: %w", key.viewName, clustSchema, n, ErrFieldIsEmpty)
			}
		}
	}
	// if err = key.clustRow.schema.validRow(&key.clustRow); err != nil {…} — unnecessary check, already checked in for-loop above

	return nil
}

// validViewValue validates specified view value
func (cache *SchemasCacheType) validViewValue(value *valueType) (err error) {
	valSchema := value.valueSchema()
	if value.QName() != valSchema {
		return validateErrorf(ECode_InvalidSchemaName, "wrong view value schema «%v», for view «%v» expected «%v»: %w", value.QName(), value.viewName, valSchema, ErrWrongSchema)
	}

	if err = value.schema.validRow(&value.rowType); err != nil {
		return err
	}

	return nil
}

// validRecord validates specified record. If rawID then raw IDs is required
func (cache *SchemasCacheType) validRecord(rec *recordType, rawID bool) (err error) {
	if rec.QName() == istructs.NullQName {
		return validateErrorf(ECode_EmptySchemaName, "record «%s» has empty schema name: %w", rec.Container(), ErrNameMissed)
	}

	schema := cache.schemaByName(rec.QName())
	switch schema.kind {
	case istructs.SchemaKind_GDoc, istructs.SchemaKind_CDoc, istructs.SchemaKind_ODoc, istructs.SchemaKind_WDoc,
		istructs.SchemaKind_GRecord, istructs.SchemaKind_CRecord, istructs.SchemaKind_ORecord, istructs.SchemaKind_WRecord:
		{
			err = schema.validRecord(rec, rawID)
			if err != nil {
				return err
			}
		}
	default:
		return validateErrorf(ECode_InvalidSchemaKind, "record «%s» refers to invalid schema «%v» kind «%s»: %w", rec.Container(), schema.name, shemaKindToStr[schema.kind], ErrUnexpectedShemaKind)
	}

	return nil
}

// schemaByName find schema by qname
func (cache *SchemasCacheType) schemaByName(name istructs.QName) (schema *SchemaType) {
	schema, ok := cache.schemas[name]
	if !ok {
		return nil
	}
	return schema
}

// ShemaType: schema
//   - interfaces:
//     — istructs.ISchema
type SchemaType struct {
	cache           *SchemasCacheType
	name            istructs.QName
	kind            istructs.SchemaKindType
	fields          map[string]*FieldPropsType
	fieldsOrder     []string
	containers      map[string]*ContainerType
	containersOrder []string
	singleton       schemaSingletonType
	dynoScheme      *dynobuffers.Scheme
}

func newSchema(cache *SchemasCacheType, name istructs.QName, kind istructs.SchemaKindType) *SchemaType {
	schema := SchemaType{
		cache:           cache,
		name:            name,
		kind:            kind,
		fields:          make(map[string]*FieldPropsType),
		fieldsOrder:     make([]string, 0),
		containers:      make(map[string]*ContainerType),
		containersOrder: make([]string, 0),
		singleton:       schemaSingletonType{},
	}
	schema.makeSysFields()
	return &schema
}

// AddField adds field specified name and kind
func (sch *SchemaType) AddField(name string, kind istructs.DataKindType, required bool) *SchemaType {
	fld := newField(name, kind, required)
	_, ok := sch.fields[name]
	sch.fields[name] = &fld
	if !ok {
		sch.fieldsOrder = append(sch.fieldsOrder, name)
	}

	return sch
}

// AddVerifiedField adds verified field specified name and kind
func (sch *SchemaType) AddVerifiedField(name string, kind istructs.DataKindType, required bool, verify ...payloads.VerificationKindType) *SchemaType {
	fld := newVerifiedField(name, kind, required, verify...)
	_, ok := sch.fields[name]
	sch.fields[name] = &fld
	if !ok {
		sch.fieldsOrder = append(sch.fieldsOrder, name)
	}

	return sch
}

// AddContainer adds container specified name and occurs
func (sch *SchemaType) AddContainer(name string, schema istructs.QName, minOccurs, maxOccurs istructs.ContainerOccursType) *SchemaType {
	cont := newContainer(name, schema, minOccurs, maxOccurs)
	_, ok := sch.containers[name]
	sch.containers[name] = &cont
	if !ok {
		sch.containersOrder = append(sch.containersOrder, name)
	}

	return sch
}

// SetSingleton sets the signleton document flag for CDoc schemas. If not CDoc schema then Validate will return error
func (sch *SchemaType) SetSingleton() {
	sch.singleton.enabled = true
}

// istructs.ISchema.QName returns schema qualified name
func (sch *SchemaType) QName() istructs.QName {
	return sch.name
}

// containerQName returns schema name for specified container name
func (sch *SchemaType) containerQName(name string) istructs.QName {
	if cont, ok := sch.containers[name]; ok {
		return cont.schema
	}
	return istructs.NullQName
}

// entName return readable name of entity to validate.
// If entity has only type QName, then the result will be short like `CDoc (sales.BillDocument)`, otherwise it will be complete like `CRecord «Price» (sales.PriceRecord)`
func (sch *SchemaType) entName(e interface{}) string {
	ent := shemaKindToStr[sch.kind]
	name := ""
	typeName := sch.name

	if row, ok := e.(istructs.IRowReader); ok {
		if qName := row.AsQName(istructs.SystemField_QName); qName != istructs.NullQName {
			typeName = qName
			if (qName == sch.name) && schemaNeedSysField_Container(sch.kind) {
				if cont := row.AsString(istructs.SystemField_Container); cont != "" {
					name = cont
				}
			}
		}
	}

	if name == "" {
		return fmt.Sprintf("%s (%v)", ent, typeName) // short form
	}

	return fmt.Sprintf("%s «%s» (%v)", ent, name, typeName) // complete form
}

// makeSysFields if required by schema kind system fields (sys.ID, sys.ParentID, etc.) are not exists, then creates them
func (sch *SchemaType) makeSysFields() {
	if schemaNeedSysField_QName(sch.kind) {
		sch.AddField(istructs.SystemField_QName, istructs.DataKind_QName, true)
	}

	if schemaNeedSysField_ID(sch.kind) {
		sch.AddField(istructs.SystemField_ID, istructs.DataKind_RecordID, true)
	}

	if schemaNeedSysField_ParentID(sch.kind) {
		sch.AddField(istructs.SystemField_ParentID, istructs.DataKind_RecordID, true)
	}

	if schemaNeedSysField_Container(sch.kind) {
		sch.AddField(istructs.SystemField_Container, istructs.DataKind_string, true)
	}

	if schemaNeedSysField_IsActive(sch.kind) {
		sch.AddField(istructs.SystemField_IsActive, istructs.DataKind_bool, false)
	}
}

// validDocument validate specified document
func (sch *SchemaType) validDocument(doc *elementType) (err error) {
	if err := sch.validElement(doc, true); err != nil {
		return err
	}

	// TODO: check RecordID refs available for document kind

	return nil
}

// validObject validate specified object
func (sch *SchemaType) validObject(obj *elementType) (err error) {
	if err := sch.validElement(obj, false); err != nil {
		return err
	}
	return nil
}

// validElement validate specified element
func (sch *SchemaType) validElement(el *elementType, storable bool) (err error) {
	if storable {
		if err := sch.validRecord(&el.recordType, true); err != nil {
			return err
		}
	} else {
		if err := sch.validRow(&el.recordType.rowType); err != nil {
			return fmt.Errorf("%s has not valid row data: %w", sch.entName(el), err)
		}
	}

	if err := sch.validElementContainers(el, storable); err != nil {
		return err
	}

	return nil
}

// validElementContainers validates element part: containers
func (sch *SchemaType) validElementContainers(el *elementType, storable bool) (err error) {
	if err := sch.validElementContOccurses(el); err != nil {
		return err
	}

	elID := el.ID()

	for i, child := range el.childs {
		childName := child.Container()
		if childName == "" {
			return validateErrorf(ECode_EmptyElementName, "%s child[%d] has empty container name: %w", sch.entName(el), i, ErrNameMissed)
		}
		cont, ok := sch.containers[childName]
		if !ok {
			return validateErrorf(ECode_InvalidElementName, "%s child[%d] has unknown container name «%s»: %w", sch.entName(el), i, childName, ErrNameNotFound)
		}

		childQName := child.QName()
		if childQName != cont.schema {
			return validateErrorf(ECode_InvalidSchemaName, "%s child[%d] «%s» has wrong schema name «%v», expected «%v»: %w", sch.entName(el), i, childName, childQName, cont.schema, ErrNameNotFound)
		}

		if storable {
			parID := child.Parent()
			if parID == istructs.NullRecordID {
				child.setParent(elID) // if child parentID omitted, then restore it
			} else {
				if parID != elID {
					return validateErrorf(ECode_InvalidRefRecordID, "%s child[%d] «%s (%v)» has wrong parent id «%d», expected «%d»: %w", sch.entName(el), i, childName, childQName, elID, parID, ErrWrongRecordID)
				}
			}
		}

		childSchema := sch.cache.schemaByName(childQName)
		err = childSchema.validElement(child, storable)
		if err != nil {
			return err
		}
	}

	return nil
}

// validElementContOccurses validates view schema part: containers occurses
func (sch *SchemaType) validElementContOccurses(el *elementType) (err error) {
	for contName, cont := range sch.containers {
		occurs := istructs.ContainerOccursType(0)
		for _, child := range el.childs {
			if child.Container() == contName {
				occurs++
			}
		}
		if occurs < cont.minOccurs {
			return validateErrorf(ECode_InvalidOccursMin, "%s container «%s» has not enough occurrences (%d, minimum %d): %w", sch.entName(el), contName, occurs, cont.minOccurs, ErrMinOccursViolation)
		}
		if occurs > cont.maxOccurs {
			return validateErrorf(ECode_InvalidOccursMax, "%s container «%s» has too many occurrences (%d, maximum %d): %w", sch.entName(el), contName, occurs, cont.maxOccurs, ErrMaxOccursViolation)
		}
	}
	return nil
}

// validRecord validates specified record. If rawID then raw IDs is required
func (sch *SchemaType) validRecord(rec *recordType, rawID bool) (err error) {
	err = sch.validRow(&rec.rowType)
	if err != nil {
		return err
	}

	if schemaNeedSysField_ID(sch.kind) {
		if rawID && !rec.ID().IsRaw() {
			return validateErrorf(ECode_InvalidRawRecordID, "new %s ID «%d» is not raw: %w", sch.entName(rec), rec.ID(), ErrRawRecordIDExpected)
		}
	}

	// if schemaNeedSysField_ID(sch.kind) && (rec.ID() == istructs.NullRecordID) {…} — unnecessary check, must be checked in sch.validRow()
	// if schemaNeedSysField_ParentID(sch.kind) && (rec.Parent() == istructs.NullRecordID) {…} — unnecessary check, must be checked in sch.validRow()
	// if schemaNeedSysField_Container(sch.kind) && ( rec.Container() == "") {…} — unnecessary check, must be checked in sch.validRow()

	return nil
}

// validRow validates specified row
func (sch *SchemaType) validRow(row *rowType) error {
	for n, fld := range sch.fields {
		if fld.required {
			if !row.hasValue(n) {
				return validateErrorf(ECode_EmptyData, "%s misses field «%s» required by schema «%v»: %w", sch.entName(row), n, sch.name, ErrNameNotFound)
			}
		}
	}

	return nil
}

// istructs.ISchema.Kind
func (sch *SchemaType) Kind() istructs.SchemaKindType {
	return sch.kind
}

// istructs.ISchema.Fields
func (sch *SchemaType) Fields(cb func(fieldName string, kind istructs.DataKindType)) {
	for _, n := range sch.fieldsOrder {
		f := sch.fields[n]
		cb(n, f.kind)
	}
}

// istructs.ISchema.ForEachField
func (sch *SchemaType) ForEachField(cb func(istructs.IFieldDescr)) {
	for _, n := range sch.fieldsOrder {
		f := sch.fields[n]
		cb(f)
	}
}

// istructs.ISchema.Containers
func (sch *SchemaType) Containers(cb func(containerName string, schema istructs.QName)) {
	for _, n := range sch.containersOrder {
		c := sch.containers[n]
		cb(n, c.schema)
	}
}

// istructs.ISchema.ForEachContainer
func (sch *SchemaType) ForEachContainer(cb func(istructs.IContainerDescr)) {
	for _, n := range sch.containersOrder {
		c := sch.containers[n]
		cb(c)
	}
}

// ViewSchemaType service view schema cortage
type ViewSchemaType struct {
	name                                             istructs.QName
	viewSchema, partSchema, clustSchema, valueSchema *SchemaType
}

func newViewSchema(cache *SchemasCacheType, name istructs.QName) ViewSchemaType {

	const (
		pk  = "_PartitionKey"
		cc  = "_ClusteringColumns"
		val = "_Value"
	)

	qNameSuffix := func(suff string) istructs.QName {
		return istructs.NewQName(name.Pkg(), name.Entity()+suff)
	}

	view := ViewSchemaType{
		name:        name,
		viewSchema:  cache.Add(name, istructs.SchemaKind_ViewRecord),
		partSchema:  cache.Add(qNameSuffix(pk), istructs.SchemaKind_ViewRecord_PartitionKey),
		clustSchema: cache.Add(qNameSuffix(cc), istructs.SchemaKind_ViewRecord_ClusteringColumns),
		valueSchema: cache.Add(qNameSuffix(val), istructs.SchemaKind_ViewRecord_Value),
	}
	view.viewSchema.
		AddContainer(istructs.SystemContainer_ViewPartitionKey, view.partSchema.name, 1, 1).
		AddContainer(istructs.SystemContainer_ViewClusteringCols, view.clustSchema.name, 1, 1).
		AddContainer(istructs.SystemContainer_ViewValue, view.valueSchema.name, 1, 1)

	return view
}

// AddPartField adds specisified field to view partition key schema. Fields is always required
func (view *ViewSchemaType) AddPartField(name string, kind istructs.DataKindType) *ViewSchemaType {
	view.partSchema.AddField(name, kind, true)
	return view
}

// AddClustColumn adds specisified field to view clustering columns schema. Fields is optional
func (view *ViewSchemaType) AddClustColumn(name string, kind istructs.DataKindType) *ViewSchemaType {
	view.clustSchema.AddField(name, kind, false)
	return view
}

// AddValueField adds specisified field to view value schema
func (view *ViewSchemaType) AddValueField(name string, kind istructs.DataKindType, required bool) *ViewSchemaType {
	view.valueSchema.AddField(name, kind, required)
	return view
}

// Name returns view name
func (view *ViewSchemaType) Name() istructs.QName {
	return view.name
}

// Schema returns view schema
func (view *ViewSchemaType) Schema() *SchemaType {
	return view.viewSchema
}

// PartKeySchema: returns view partition key schema
func (view *ViewSchemaType) PartKeySchema() *SchemaType {
	return view.partSchema
}

// ClustColsSchema returns view clustering columns schema
func (view *ViewSchemaType) ClustColsSchema() *SchemaType {
	return view.clustSchema
}

// ValueSchema returns view value schema
func (view *ViewSchemaType) ValueSchema() *SchemaType {
	return view.valueSchema
}

// FieldPropsType is description of single field type
// implement istructs.IFieldDescr interface
type FieldPropsType struct {
	name       string
	kind       istructs.DataKindType
	required   bool
	verifiable bool
	verify     map[payloads.VerificationKindType]bool
}

func newField(name string, kind istructs.DataKindType, required bool) FieldPropsType {
	return FieldPropsType{name, kind, required, false, make(map[payloads.VerificationKindType]bool)}
}

func newVerifiedField(name string, kind istructs.DataKindType, required bool, verify ...payloads.VerificationKindType) FieldPropsType {
	ft := FieldPropsType{name, kind, required, true, make(map[payloads.VerificationKindType]bool)}
	for _, kind := range verify {
		ft.verify[kind] = true
	}
	return ft
}

// istructs.IFieldDescr.Name()
func (fld *FieldPropsType) Name() string { return fld.name }

// istructs.IFieldDescr.DataKind()
func (fld *FieldPropsType) DataKind() istructs.DataKindType { return fld.kind }

// istructs.IFieldDescr.Required()
func (fld *FieldPropsType) Required() bool { return fld.required }

// istructs.IFieldDescr.Verifiable()
func (fld *FieldPropsType) Verifiable() bool { return fld.verifiable }

// ContainerType occurs of nested schema part
// implements istructs.IContainerDesc interface
type ContainerType struct {
	name      string
	schema    istructs.QName
	minOccurs istructs.ContainerOccursType
	maxOccurs istructs.ContainerOccursType
}

func newContainer(name string, schema istructs.QName, minOccurs, maxOccurs istructs.ContainerOccursType) ContainerType {
	return ContainerType{
		name:      name,
		schema:    schema,
		minOccurs: minOccurs,
		maxOccurs: maxOccurs,
	}
}

// istructs.IContainerDesc.Name()
func (cont *ContainerType) Name() string { return cont.name }

// istructs.IContainerDesc.Schema()
func (cont *ContainerType) Schema() istructs.QName { return cont.schema }

// istructs.IContainerDesc.MinOccurs()
func (cont *ContainerType) MinOccurs() istructs.ContainerOccursType { return cont.minOccurs }

// istructs.IContainerDesc.MaxOccurs()
func (cont *ContainerType) MaxOccurs() istructs.ContainerOccursType { return cont.maxOccurs }

// schemaSingletonType is type for cdoc schema singleton params
type schemaSingletonType struct {
	enabled bool
	id      istructs.RecordID
}
