/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/uniques"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func getUniqueNumber(vit *it.VIT) (int, string) {
	uniqueNumber := vit.NextNumber()
	buf := bytes.NewBuffer(nil)
	require.NoError(vit.T, binary.Write(buf, binary.BigEndian, uint32(uniqueNumber)))
	uniqueBytes := base64.StdEncoding.EncodeToString(buf.Bytes())
	return uniqueNumber, uniqueBytes
}

func TestBasicUsage_Uniques(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	vit.PostWS(ws, "c.sys.CUD", body)

	t.Run("409 on duplicate basic", func(t *testing.T) {
		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409()).Println()
	})

	t.Run("409 on duplicate different fields order", func(t *testing.T) {
		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Str":"str","Bytes":"%s","Int":%d,"Bool":true}}]}`, bts, num)
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409()).Println()
	})
}

func TestActivateDeactivateRecordWithUniques(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	// insert a unique
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	newID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// allowed to deactivate
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":false}}]}`, newID)
	vit.PostWS(ws, "c.sys.CUD", body)

	// ok to deactivate again an already inactive record
	vit.PostWS(ws, "c.sys.CUD", body)

	// still not able to update unique fields even if the record is deactivated
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Int":42}}]}`, newID)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403())

	// allowed to activate
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":true}}]}`, newID)
	vit.PostWS(ws, "c.sys.CUD", body)

	// ok to activate again the already active record
	vit.PostWS(ws, "c.sys.CUD", body)

	// check uniques works after deactivate/activate
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
}

func TestUniquesUpdate(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	// insert a unique
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	prevID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// to update unique fields let's deactivate existing record and create new record with new values
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":false}}]}`, prevID)
	vit.PostWS(ws, "c.sys.CUD", body)

	// insert a record with new values, i.e. do actually update
	num, bts = getUniqueNumber(vit)
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	prevID = vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// let's deactivate the new record
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":false}}]}`, prevID)
	vit.PostWS(ws, "c.sys.CUD", body)

	// we're able to insert a new record that conflicts with the deactivated one
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	vit.PostWS(ws, "c.sys.CUD", body)

	// insert the same again -> unique constraint violation with the new record
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())

	// try to activate previously deactivated record -> conflict with the just inserted new record
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":true}}]}`, prevID)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
}

func TestUniquesDenyUpdate(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	// insert one unique
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	newID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// deny to modify any unique field
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","Int": 1}}]}`, newID)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403())
}

func TestInsertDeactivatedRecord(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	// insert a deactivated unique
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":false,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	newID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// still not able to update unique fields even if the record is deactivated
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"Int":42}}]}`, newID)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403())

	// allowed to activate
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":true}}]}`, newID)
	vit.PostWS(ws, "c.sys.CUD", body)

	// check uniques works after deactivate/activate
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())

	// allowed to insert a new deactivated record event it is conflicting with existsing
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":false,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	vit.PostWS(ws, "c.sys.CUD", body)
}

func TestUniquesTrickyValues(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("order of fields should not cause violation", func(t *testing.T) {
		// row with int32 12345678,[]byte{0} are inserted
		// have unique combination for key []byte{7, 91, 205, 21, 0}
		// insert a new row for []byte{7}, int32([]byte{91, 205, 21, 0})
		// should not cause violation
		// the test protects the rule that var size field must be used last
		num := 123456789 // []byte{7, 91, 205, 21}
		bts := base64.StdEncoding.EncodeToString([]byte{0})
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":false,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
		vit.PostWS(ws, "c.sys.CUD", body)

		// have unique for key {7,91,205,21,0}

		num = int(binary.BigEndian.Uint32([]byte{91, 205, 21, 0}))
		bts = base64.StdEncoding.EncodeToString([]byte{7})
		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":false,"Bytes":"%s","Int":%d,"Str":"str","Bool":true}}]}`, bts, num)
		vit.PostWS(ws, "c.sys.CUD", body)
		// expect no errors
	})
}

func TestMultipleCUDs(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	t.Run("duplicate in cuds simple", func(t *testing.T) {
		t.Run("duplicate with existing", func(t *testing.T) {
			body := fmt.Sprintf(`{"cuds":[
				{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}},
				{"fields":{"sys.ID":2,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}
			]}`, num, bts, num, bts)
			vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
		})

		t.Run("duplicate in CUDs", func(t *testing.T) {
			newNum, newBts := getUniqueNumber(vit)
			body := fmt.Sprintf(`{"cuds":[
				{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}},
				{"fields":{"sys.ID":2,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}
			]}`, newNum, newBts, newNum, newBts)
			vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
		})
	})

	t.Run("multiple update", func(t *testing.T) {
		t.Run("update the inactive record", func(t *testing.T) {
			num, bts := getUniqueNumber(vit)
			body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
			id := vit.PostWS(ws, "c.sys.CUD", body).NewID()

			t.Run("any CUD itself produces the conflict -> 409 even if effectively no conflict", func(t *testing.T) {
				t.Run("produce the conflict by insert, remove the conflict by deactivate existing", func(t *testing.T) {

					// 1st cud should produce the conflict but the 2nd should deactivate the existing record, i.e. effectively there should not be the conflict
					// but the batch is not atomic so it is possible connection with the storage between 1st and 2nd CUDs that will produce the unique violation in the storage
					// so that should be denied
					body = fmt.Sprintf(`{"cuds":[
						{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}},
						{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":false}}
						]}`, num, bts, id)
					vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
				})

				t.Run("insert non-conflicting, produce the conflict by activating the existing inactive record", func(t *testing.T) {
					num, bts := getUniqueNumber(vit)

					// insert one inactive
					body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":false,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
					id := vit.PostWS(ws, "c.sys.CUD", body).NewID()

					// insert new same (ok) and update the incative one (should be denied)
					body = fmt.Sprintf(`{"cuds":[
						{"fields":{"sys.ID":2,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}},
						{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":true}}
					]}`, num, bts, id)
					vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
				})
			})

			t.Run("effectively no changes", func(t *testing.T) {
				body = fmt.Sprintf(`{"cuds":[
					{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":false}},
					{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":true}}
				]}`, id, id)
				vit.PostWS(ws, "c.sys.CUD", body)
			})

			t.Run("effectively no changes but insert a conflicting record", func(t *testing.T) {
				body = fmt.Sprintf(`{"cuds":[
					{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":false}},
					{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":true}},
					{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}
				]}`, id, id, num, bts)
				vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
			})
		})
		t.Run("update the inactive record", func(t *testing.T) {
			num, bts := getUniqueNumber(vit)

			body := fmt.Sprintf(`{"cuds":[
				{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":false,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}},
				{"fields":{"sys.ID":2,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}
			]}`, num, bts, num, bts)
			resp := vit.PostWS(ws, "c.sys.CUD", body)
			id1 := resp.NewIDs["1"]

			t.Run("ok to make first conflict by update then fix it immediately because engine stores updates in a map by ID, result is false in our case", func(t *testing.T) {
				// cmd.reb.cud.updates is map, so first we set one ID isActive=true that shout make it conflict
				// but on creating the update for the secod CUD we overwrite the map by the same ID and write SetActive=false
				// see commandprocessor.writeCUDs()
				body = fmt.Sprintf(`{"cuds":[
					{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":true}},
					{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":false}}
				]}`, id1, id1)
				vit.PostWS(ws, "c.sys.CUD", body)
			})

			t.Run("conflict if effectively activating the conflicting record", func(t *testing.T) {
				body = fmt.Sprintf(`{"cuds":[
					{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":false}},
					{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":true}}
				]}`, id1, id1)
				vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
			})
		})

	})
}

func TestBasicUsage_GetUniqueID(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	// insert a doc record that has an unique
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.DocConstraints","sys.IsActive":true,"Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	newID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	as, err := vit.IAppStructsProvider.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)

	t.Run("must be ok to find unique field for test CDoc", func(t *testing.T) {
		unique, ok := as.AppDef().Def(it.QNameCDocTestConstraints).(appdef.IUniques)
		require.True(ok)
		require.NotNil(unique.UniqueField())
	})

	// simulate data source and try to get an ID for that combination of key fields
	t.Run("basic", func(t *testing.T) {
		obj := &coreutils.TestObject{
			Data: map[string]interface{}{
				// required for unique key builder
				appdef.SystemField_QName: it.QNameCDocTestConstraints,
				// required for unique key
				"Int": int32(num),
				// not in the unique key, could be omitted
				"Str":     "str",
				"Bool":    true,
				"Float32": float32(42),
			},
		}
		uniqueRecID, err := uniques.GetUniqueID(as, obj, ws.WSID)
		require.NoError(err)
		require.Equal(istructs.RecordID(newID), uniqueRecID)
	})

	t.Run("must be ok to deactivate active record", func(t *testing.T) {
		// let's deactivate the record
		body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":false}}]}`, newID)
		vit.PostWS(ws, "c.sys.CUD", body)

		t.Run("must be not found deactivated record", func(t *testing.T) {
			// NullRecordID for the inactive record
			obj := &coreutils.TestObject{
				Data: map[string]interface{}{
					appdef.SystemField_QName: it.QNameCDocTestConstraints,
					"Int":                    int32(num),
					"Str":                    "str",
					"Bool":                   true,
				},
			}
			uniqueRecID, err := uniques.GetUniqueID(as, obj, ws.WSID)
			require.NoError(err)
			require.Zero(uniqueRecID)
		})
	})

	t.Run("must be not found unknown record", func(t *testing.T) {
		obj := &coreutils.TestObject{
			Data: map[string]interface{}{
				appdef.SystemField_QName: it.QNameCDocTestConstraints,
				"Int":                    int32(num) + 1,
			},
		}
		uniqueRecID, err := uniques.GetUniqueID(as, obj, ws.WSID)
		require.NoError(err)
		require.Zero(uniqueRecID)
	})

	t.Run("must be ok to reactivate inactive record", func(t *testing.T) {
		// let's reactivate the record
		body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"test.DocConstraints","sys.IsActive":true}}]}`, newID)
		vit.PostWS(ws, "c.sys.CUD", body)

		t.Run("must be ok reactivated record", func(t *testing.T) {
			obj := &coreutils.TestObject{
				Data: map[string]interface{}{
					appdef.SystemField_QName: it.QNameCDocTestConstraints,
					"Int":                    int32(num),
					"Str":                    "str",
					"Bool":                   true,
				},
			}
			uniqueRecID, err := uniques.GetUniqueID(as, obj, ws.WSID)
			require.NoError(err)
			require.Equal(istructs.RecordID(newID), uniqueRecID)
		})
	})
}
