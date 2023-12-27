/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
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
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	vit.PostWS(ws, "c.sys.CUD", body)

	t.Run("409 on duplicate basic", func(t *testing.T) {
		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409()).Println()
	})

	t.Run("409 on duplicate different fields order", func(t *testing.T) {
		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Str":"str","Bytes":"%s","Int":%d,"Bool":true}}]}`, bts, num)
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409()).Println()
	})
}

func TestActivateDeactivateRecordWithUniques(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	// insert a unique
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
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
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
}

func TestUniquesUpdate(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	// insert a unique
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	prevID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// to update unique fields let's deactivate existing record and create new record with new values
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":false}}]}`, prevID)
	vit.PostWS(ws, "c.sys.CUD", body)

	// insert a record with new values, i.e. do actually update
	num, bts = getUniqueNumber(vit)
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	prevID = vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// let's deactivate the new record
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":false}}]}`, prevID)
	vit.PostWS(ws, "c.sys.CUD", body)

	// we're able to insert a new record that conflicts with the deactivated one
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	vit.PostWS(ws, "c.sys.CUD", body)

	// insert the same again -> unique constraint violation with the new record
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())

	// try to activate previously deactivated record -> conflict with the just inserted new record
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.IsActive":true}}]}`, prevID)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
}

func TestUniquesDenyUpdate(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	// insert one unique
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
	newID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// deny to modify any unique field
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","Int": 1}}]}`, newID)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403())
}

func TestFewUniquesOneDoc(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("same sets of values for 2 different uniques in one doc -> no coflict", func(t *testing.T) {
		newNum, newBts := getUniqueNumber(vit)
		// the doc has 2 uniques, Bytes1 fields belongs to both
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraintsFewUniques",
			"Int1":%[1]d,"Str1":"str","Bool1":true,"Bytes1":"%[2]s",
			"Int2":%[1]d,"Str2":"str","Bool2":true,"Bytes1":"%[2]s"}}]}`, newNum, newBts)
		vit.PostWS(ws, "c.sys.CUD", body)
	})

	t.Run("no values for unique fields -> default values are used", func(t *testing.T) {
		t.Run("uniq1", func(t *testing.T) {

			// insert a record with no values for unique fields -> default values are used as unique values
			// have values for uniq2, no values for uniq1
			newNum, _ := getUniqueNumber(vit)
			body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraintsFewUniques",
				"Int2":%d,"Str2":"str","Bool2":true}}]}`, newNum)
			vit.PostWS(ws, "c.sys.CUD", body)

			// no values again -> conflict because unique combination for default values exists already
			// have values for uniq2, no values for uniq1
			newNum, _ = getUniqueNumber(vit)
			body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraintsFewUniques",
				"Int2":%d,"Str2":"str","Bool2":true}}]}`, newNum)
			vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409("uniq1")).Println()
		})
		t.Run("uniq2", func(t *testing.T) {

			// insert a record with no values for unique fields -> default values are used as unique values
			// have values for uniq1, no values for uniq2
			newNum, _ := getUniqueNumber(vit)
			body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraintsFewUniques",
				"Int1":%d,"Str1":"str","Bool1":true}}]}`, newNum)
			vit.PostWS(ws, "c.sys.CUD", body)

			// no values again -> conflict because unique combination for default values exists already
			// have values for uniq1, no values for uniq2
			newNum, _ = getUniqueNumber(vit)
			body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraintsFewUniques",
				"Int1":%d,"Str1":"str","Bool1":true}}]}`, newNum)
			vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409("uniq2")).Println()
		})
	})
}

func TestMultipleCUDs(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("duplicate in cuds", func(t *testing.T) {
		newNum, newBts := getUniqueNumber(vit)
		body := fmt.Sprintf(`{"cuds":[
				{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}},
				{"fields":{"sys.ID":2,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}
			]}`, newNum, newBts, newNum, newBts)
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
	})

	t.Run("update few records in one request", func(t *testing.T) {
		t.Run("update the inactive record", func(t *testing.T) {
			num, bts := getUniqueNumber(vit)
			// insert the first record
			body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
			id := vit.PostWS(ws, "c.sys.CUD", body).NewID()

			t.Run("any CUD itself produces the conflict -> 409 even if effectively no conflict", func(t *testing.T) {
				t.Run("produce the conflict by insert, remove the conflict by deactivate existing", func(t *testing.T) {

					// 1st cud should produce the conflict but the 2nd should deactivate the existing record, i.e. effectively there should not be the conflict
					// but the batch is not atomic so it is possible connection with the storage between 1st and 2nd CUDs that will produce the unique violation in the storage
					// so that should be denied
					body = fmt.Sprintf(`{"cuds":[
						{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}},
						{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":false}}
						]}`, num, bts, id)
					vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
				})

				t.Run("insert non-conflicting, produce the conflict by activating the existing inactive record", func(t *testing.T) {
					num, bts := getUniqueNumber(vit)

					// insert one record
					body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
					id := vit.PostWS(ws, "c.sys.CUD", body).NewID()

					// deactivate it
					body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":false}}]}`, id)
					vit.PostWS(ws, "c.sys.CUD", body)

					// insert new same (ok) and activate the incative one -> deny
					body = fmt.Sprintf(`{"cuds":[
						{"fields":{"sys.ID":2,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}},
						{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":true}}
					]}`, num, bts, id)
					vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
				})
			})

			t.Run("effectively no changes and insert a conflicting record", func(t *testing.T) {
				body = fmt.Sprintf(`{"cuds":[
					{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":false}},
					{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":true}},
					{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}
				]}`, id, id, num, bts)
				vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
			})
		})
		t.Run("update the inactive record", func(t *testing.T) {
			num, bts := getUniqueNumber(vit)

			// insert one record
			body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
			id1 := vit.PostWS(ws, "c.sys.CUD", body).NewID()

			// deactivate it
			body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":false}}]}`, id1)
			vit.PostWS(ws, "c.sys.CUD", body)

			// insert one more the same record
			body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
			vit.PostWS(ws, "c.sys.CUD", body)

			t.Run("ok to make first conflict by update then fix it immediately because engine stores updates in a map by ID, result is false in our case", func(t *testing.T) {
				// cmd.reb.cud.updates is map, so first we set one ID isActive=true that shout make a conflict
				// but on creating the update for the second CUD we overwrite the map by the same ID and write SetActive=false
				// i.e. update the same record twice -> one update operation with merged fields, result is deactivation
				// see commandprocessor.writeCUDs()
				body = fmt.Sprintf(`{"cuds":[
					{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":true}},
					{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":false}}
				]}`, id1, id1)
				vit.PostWS(ws, "c.sys.CUD", body)
			})

			t.Run("conflict if effectively activating the conflicting record", func(t *testing.T) {
				// update the same record twice -> one update operation with merged fields, result is activation
				body = fmt.Sprintf(`{"cuds":[
					{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":false}},
					{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":true}}
				]}`, id1, id1)
				vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409())
			})
		})

	})
}

func TestBasicUsage_GetUniqueRecordID(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	as, err := vit.IAppStructsProvider.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)

	t.Run("basic", func(t *testing.T) {
		// insert a doc record that has an unique
		num, bts := getUniqueNumber(vit)
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
		newID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

		bytes, err := base64.StdEncoding.DecodeString(bts)
		require.NoError(err)
		obj := &coreutils.TestObject{
			Data: map[string]interface{}{
				// required for unique key builder
				appdef.SystemField_QName: it.QNameApp1_DocConstraints,
				// required for unique key
				"Int":   int32(num),
				"Str":   "str",
				"Bool":  true,
				"Bytes": bytes,
				// not in the unique key, could be omitted
				"Float32": float32(42),
			},
		}
		uniqueRecID, err := uniques.GetUniqueRecordID(as, obj, ws.WSID)
		require.NoError(err)
		require.Equal(istructs.RecordID(newID), uniqueRecID)
	})

	t.Run("deactivated and reactivated record", func(t *testing.T) {
		// insert a doc record that has an unique
		num, bts := getUniqueNumber(vit)
		bytes, err := base64.StdEncoding.DecodeString(bts)
		require.NoError(err)
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, num, bts)
		newID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

		// let's deactivate the record
		body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":false}}]}`, newID)
		vit.PostWS(ws, "c.sys.CUD", body)

		t.Run("NullRecordID on the deactivated record", func(t *testing.T) {
			// NullRecordID for the inactive record
			obj := &coreutils.TestObject{
				Data: map[string]interface{}{
					appdef.SystemField_QName: it.QNameApp1_DocConstraints,
					"Int":                    int32(num),
					"Str":                    "str",
					"Bool":                   true,
					"Bytes":                  bytes,
				},
			}
			uniqueRecID, err := uniques.GetUniqueRecordID(as, obj, ws.WSID)
			require.NoError(err)
			require.Zero(uniqueRecID)
		})

		body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"app1pkg.DocConstraints","sys.IsActive":true}}]}`, newID)
		vit.PostWS(ws, "c.sys.CUD", body)

		t.Run("valid RecordID on the reactivated record", func(t *testing.T) {
			obj := &coreutils.TestObject{
				Data: map[string]interface{}{
					appdef.SystemField_QName: it.QNameApp1_DocConstraints,
					"Int":                    int32(num),
					"Str":                    "str",
					"Bool":                   true,
					"Bytes":                  bytes,
				},
			}
			uniqueRecID, err := uniques.GetUniqueRecordID(as, obj, ws.WSID)
			require.NoError(err)
			require.Equal(istructs.RecordID(newID), uniqueRecID)
		})
	})

	t.Run("NullRecordID on an unexisting unique combination", func(t *testing.T) {
		num, _ := getUniqueNumber(vit)
		obj := &coreutils.TestObject{
			Data: map[string]interface{}{
				appdef.SystemField_QName: it.QNameApp1_DocConstraints,
				"Int":                    int32(num) + 1,
			},
		}
		uniqueRecID, err := uniques.GetUniqueRecordID(as, obj, ws.WSID)
		require.NoError(err)
		require.Zero(uniqueRecID)
	})
}

func TestMaxUniqueLen(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	const bigLen = 42000

	num, _ := getUniqueNumber(vit)
	longString := strings.Repeat(" ", bigLen)
	longBytes := make([]byte, bigLen)
	longBytesBase64 := base64.StdEncoding.EncodeToString(longBytes)
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraints","Int":%d,"Str":"%s","Bool":true,"Bytes":"%s"}}]}`, num, longString, longBytesBase64)
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403(uniques.ErrUniqueValueTooLong.Error())).Println()
}

func TestBasicUsage_UNIQUEFIELD(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	num, bts := getUniqueNumber(vit)

	// insert an initial record
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraintsOldAndNewUniques","Int":%d,"Str":"%s"}}]}`, num, string(bts))
	vit.PostWS(ws, "c.sys.CUD", body)

	// fire the UNIQUEFIELD violation, avoid UNIQUE (Str) vioaltion
	_, newBts := getUniqueNumber(vit)
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocConstraintsOldAndNewUniques","Int":%d,"Str":"%s"}}]}`, num, string(newBts))
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect409("UNIQUEFIELD Int"))
}
