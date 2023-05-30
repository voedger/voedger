/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"context"
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/sys/smtp"

	"github.com/voedger/voedger/pkg/appdef"
	registryapp "github.com/voedger/voedger/pkg/apps/sys/registry"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz/wskinds"
	sys_test_template "github.com/voedger/voedger/pkg/vit/testdata"
	"github.com/voedger/voedger/pkg/vvm"
)

const (
	TestEmail        = "123@123.com"
	TestEmail2       = "124@124.com"
	TestEmail3       = "125@125.com"
	TestServicePort  = 10000
	defaultMaxOccurs = 100
)

var (
	QNameTestWSKind               = appdef.NewQName("my", "WSKind")
	QNameTestTable                = appdef.NewQName("sys", "air_table_plan")
	QNameTestView                 = appdef.NewQName("my", "View")
	QNameTestEmailVerificationDoc = appdef.NewQName("test", "Doc")
	QNameCDocTestConstraints      = appdef.NewQName("test", "DocConstraints")
	QNameTestSingleton            = appdef.NewQName("test", "Config")
	QNameCmdRated                 = appdef.NewQName(appdef.SysPackage, "RatedCmd")
	QNameQryRated                 = appdef.NewQName(appdef.SysPackage, "RatedQry")

	// BLOBMaxSize 5
	SharedConfig_Simple = NewSharedVITConfig(
		WithApp(istructs.AppQName_test1_app1, ProvideSimpleApp,
			WithWorkspaceTemplate(QNameTestWSKind, "test_template", sys_test_template.TestTemplateFS),
			WithUserLogin("login", "pwd"),
			WithUserLogin(TestEmail, "1"),
			WithUserLogin(TestEmail2, "1"),
			WithUserLogin(TestEmail3, "1"),
			WithChildWorkspace(QNameTestWSKind, "test_ws", "test_template", "", "login", map[string]interface{}{"IntFld": 42}),
		),
		WithApp(istructs.AppQName_test1_app2, ProvideSimpleApp, WithUserLogin("login", "1")),
		WithVVMConfig(func(cfg *vvm.VVMConfig) {
			// for impl_reverseproxy_test
			cfg.Routes["/grafana"] = fmt.Sprintf("http://127.0.0.1:%d", TestServicePort)
			cfg.RoutesRewrite["/grafana-rewrite"] = fmt.Sprintf("http://127.0.0.1:%d/rewritten", TestServicePort)
			cfg.RouteDefault = fmt.Sprintf("http://127.0.0.1:%d/not-found", TestServicePort)
			cfg.RouteDomains["localhost"] = "http://127.0.0.1"
		}),
		WithCleanup(func(_ *VIT) {
			MockCmdExec = func(input string) error { panic("") }
			MockQryExec = func(input string, callback istructs.ExecQueryCallback) error { panic("") }
		}),
	)
	MockQryExec func(input string, callback istructs.ExecQueryCallback) error
	MockCmdExec func(input string) error
)

func EmptyApp(vvmCfg *vvm.VVMConfig, vvmAPI vvm.VVMAPI, cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, sep vvm.IStandardExtensionPoints) {
	registryapp.Provide(smtp.Cfg{})(vvmCfg, vvmAPI, cfg, adf, sep)
	adf.AddSingleton(QNameTestWSKind).
		AddField("IntFld", appdef.DataKind_int32, true).
		AddField("StrFld", appdef.DataKind_string, false)
	sep.ExtensionPoint(wskinds.EPWorkspaceKind).Add(QNameTestWSKind)
}

func ProvideSimpleApp(vvmCfg *vvm.VVMConfig, vvmAPI vvm.VVMAPI, cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, sep vvm.IStandardExtensionPoints) {

	// sys package
	sys.Provide(vvmCfg.TimeFunc, cfg, adf, vvmAPI, smtp.Cfg{}, sep, nil)

	const simpleAppBLOBMaxSize = 5
	vvmCfg.BLOBMaxSize = simpleAppBLOBMaxSize

	adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "articles")).
		AddField("name", appdef.DataKind_string, false).
		AddField("article_manual", appdef.DataKind_int32, true).
		AddField("article_hash", appdef.DataKind_int32, true).
		AddField("hideonhold", appdef.DataKind_int32, true).
		AddField("time_active", appdef.DataKind_int32, true).
		AddField("control_active", appdef.DataKind_int32, true)

	adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "options"))

	dep := adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "department"))
	dep.AddField("pc_fix_button", appdef.DataKind_int32, true).
		AddField("rm_fix_button", appdef.DataKind_int32, true).
		AddField("id_food_group", appdef.DataKind_RecordID, false)
	dep.
		AddContainer("department_options", appdef.NewQName(appdef.SysPackage, "department_options"), appdef.Occurs(0), appdef.Occurs(defaultMaxOccurs))

	adf.AddCRecord(appdef.NewQName(appdef.SysPackage, "department_options")).
		AddField("id_department", appdef.DataKind_RecordID, true).
		AddField("id_options", appdef.DataKind_RecordID, false).
		AddField("option_number", appdef.DataKind_int32, false).
		AddField("option_type", appdef.DataKind_int32, true)

	tabPlan := adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "air_table_plan"))
	tabPlan.
		AddField("fstate", appdef.DataKind_int32, false).
		AddField("name", appdef.DataKind_string, false).
		AddField("ml_name", appdef.DataKind_bytes, false).
		AddField("num", appdef.DataKind_int32, false).
		AddField("width", appdef.DataKind_int32, false).
		AddField("height", appdef.DataKind_int32, false).
		AddField("image", appdef.DataKind_int64, false).
		AddField("is_hidden", appdef.DataKind_int32, false).
		AddField("sys.IsActive", appdef.DataKind_bool, false).
		AddField("preview", appdef.DataKind_int64, false).
		AddField("bg_color", appdef.DataKind_int32, false)
	tabPlan.
		AddContainer("air_table_plan_item", appdef.NewQName(appdef.SysPackage, "air_table_plan_item"), appdef.Occurs(0), appdef.Occurs(defaultMaxOccurs))

	adf.AddCRecord(appdef.NewQName(appdef.SysPackage, "air_table_plan_item")).
		AddField("id_air_table_plan", appdef.DataKind_RecordID, true).
		AddField("fstate", appdef.DataKind_int32, false).
		AddField("number", appdef.DataKind_int32, false).
		AddField("form", appdef.DataKind_int32, true).
		AddField("top_c", appdef.DataKind_int32, false).
		AddField("left_c", appdef.DataKind_int32, false).
		AddField("angle", appdef.DataKind_int32, false).
		AddField("width", appdef.DataKind_int32, false).
		AddField("height", appdef.DataKind_int32, false).
		AddField("places", appdef.DataKind_int32, false).
		AddField("chair_type", appdef.DataKind_string, false).
		AddField("table_type", appdef.DataKind_string, false).
		AddField("sys.IsActive", appdef.DataKind_bool, false).
		AddField("type", appdef.DataKind_int32, false).
		AddField("color", appdef.DataKind_int32, false).
		AddField("code", appdef.DataKind_string, false).
		AddField("text", appdef.DataKind_string, false).
		AddField("hide_seats", appdef.DataKind_bool, false)

	adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "printers")).
		AddField("guid", appdef.DataKind_string, true).
		AddField("name", appdef.DataKind_string, false).
		AddField("id_printer_drivers", appdef.DataKind_RecordID, false).
		AddField("width", appdef.DataKind_int32, false).
		AddField("top_lines", appdef.DataKind_int32, false).
		AddField("bottom_lines", appdef.DataKind_int32, false).
		AddField("con", appdef.DataKind_int32, false).
		AddField("port", appdef.DataKind_int32, false).
		AddField("speed", appdef.DataKind_int32, false).
		AddField("backup_printer", appdef.DataKind_string, false).
		AddField("id_computers", appdef.DataKind_RecordID, false).
		AddField("error_flag", appdef.DataKind_int32, true).
		AddField("codepage", appdef.DataKind_int32, false).
		AddField("null_print", appdef.DataKind_int32, false).
		AddField("fiscal", appdef.DataKind_int32, false).
		AddField("dont_auto_open_drawer", appdef.DataKind_int32, false).
		AddField("connection_type", appdef.DataKind_int32, false).
		AddField("printer_ip", appdef.DataKind_string, false).
		AddField("printer_port", appdef.DataKind_int32, false).
		AddField("cant_be_redirected_to", appdef.DataKind_int32, false).
		AddField("sys.IsActive", appdef.DataKind_bool, false).
		AddField("com_params", appdef.DataKind_bytes, false).
		AddField("printer_type", appdef.DataKind_int32, false).
		AddField("exclude_message", appdef.DataKind_int32, false).
		AddField("driver_kind", appdef.DataKind_int32, false).
		AddField("driver_id", appdef.DataKind_string, false).
		AddField("driver_params", appdef.DataKind_bytes, false).
		AddField("check_status", appdef.DataKind_int32, false).
		AddField("id_ordermans", appdef.DataKind_RecordID, false).
		AddField("id_falcon_terminals", appdef.DataKind_RecordID, false).
		AddField("hht_printer_port", appdef.DataKind_int32, false).
		AddField("ml_name", appdef.DataKind_bytes, false).
		AddField("posprinter_driver_id", appdef.DataKind_string, false).
		AddField("posprinter_driver_params", appdef.DataKind_string, false).
		AddField("id_bill_ticket", appdef.DataKind_RecordID, false).
		AddField("id_order_ticket", appdef.DataKind_RecordID, false).
		AddField("purpose_receipt_enabled", appdef.DataKind_bool, false).
		AddField("purpose_preparation_enabled", appdef.DataKind_bool, false)

	adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "sales_area")).
		AddField("name", appdef.DataKind_string, false).
		AddField("bmanual", appdef.DataKind_int32, true).
		AddField("id_prices", appdef.DataKind_RecordID, false).
		AddField("number", appdef.DataKind_int32, false).
		AddField("close_manualy", appdef.DataKind_int32, false). // nolint
		AddField("auto_accept_reservations", appdef.DataKind_int32, false).
		AddField("only_reserved", appdef.DataKind_int32, false).
		AddField("id_prices_original", appdef.DataKind_RecordID, false).
		AddField("group_vat_level", appdef.DataKind_int32, false).
		AddField("sys.IsActive", appdef.DataKind_bool, false).
		AddField("sc", appdef.DataKind_int64, false).
		AddField("sccovers", appdef.DataKind_int32, false).
		AddField("id_scplan", appdef.DataKind_RecordID, false).
		AddField("price_dt", appdef.DataKind_int64, false).
		AddField("sa_external_id", appdef.DataKind_string, false).
		AddField("is_default", appdef.DataKind_bool, false).
		AddField("id_table_plan", appdef.DataKind_RecordID, false)

	adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "payments")).
		AddField("name", appdef.DataKind_string, false).
		AddField("kind", appdef.DataKind_int32, false).
		AddField("number", appdef.DataKind_int32, false).
		AddField("psp_model", appdef.DataKind_int32, false).
		AddField("id_bookkp", appdef.DataKind_RecordID, false).
		AddField("id_currency", appdef.DataKind_RecordID, false).
		AddField("sys.IsActive", appdef.DataKind_bool, false).
		AddField("params", appdef.DataKind_string, false).
		AddField("driver_kind", appdef.DataKind_int32, false).
		AddField("driver_id", appdef.DataKind_string, false).
		AddField("guid", appdef.DataKind_string, true).
		AddField("ml_name", appdef.DataKind_bytes, false).
		AddField("paym_external_id", appdef.DataKind_string, false)

	adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "untill_users")).
		AddField("name", appdef.DataKind_string, false).
		AddField("mandates", appdef.DataKind_bytes, false).
		AddField("user_void", appdef.DataKind_int32, true).
		AddField("user_code", appdef.DataKind_string, false).
		AddField("user_card", appdef.DataKind_string, false).
		AddField("language", appdef.DataKind_string, false).
		AddField("language_char", appdef.DataKind_int32, false).
		AddField("user_training", appdef.DataKind_int32, true).
		AddField("address", appdef.DataKind_string, false).
		AddField("id_countries", appdef.DataKind_RecordID, false).
		AddField("phone", appdef.DataKind_string, false).
		AddField("datebirth", appdef.DataKind_int64, false).
		AddField("insurance", appdef.DataKind_string, false).
		AddField("user_poscode", appdef.DataKind_string, false).
		AddField("terminal_id", appdef.DataKind_string, false).
		AddField("user_clock_in", appdef.DataKind_int32, false).
		AddField("user_poscode_remoteterm", appdef.DataKind_string, false).
		AddField("is_custom_remoteterm_poscode", appdef.DataKind_int32, false).
		AddField("sys.IsActive", appdef.DataKind_bool, false).
		AddField("id_group_users", appdef.DataKind_RecordID, false).
		AddField("tp_api_pwd", appdef.DataKind_string, false).
		AddField("firstname", appdef.DataKind_string, false).
		AddField("lastname", appdef.DataKind_string, false).
		AddField("user_transfer", appdef.DataKind_int32, false).
		AddField("personal_drawer", appdef.DataKind_int32, false).
		AddField("start_week_day", appdef.DataKind_int32, false).
		AddField("start_week_time", appdef.DataKind_int64, false).
		AddField("needcashdeclaration", appdef.DataKind_int32, false).
		AddField("smartcard_uid", appdef.DataKind_string, false).
		AddField("not_print_waiter_report", appdef.DataKind_int32, false).
		AddField("exclude_message", appdef.DataKind_int32, false).
		AddField("lefthand", appdef.DataKind_int32, false).
		AddField("login_message", appdef.DataKind_string, false).
		AddField("email", appdef.DataKind_string, false).
		AddField("number", appdef.DataKind_int32, false).
		AddField("hq_id", appdef.DataKind_string, false).
		AddField("void_number", appdef.DataKind_int32, false).
		AddField("last_update_dt", appdef.DataKind_int64, false).
		AddField("block_time_break", appdef.DataKind_int32, false).
		AddField("void_type", appdef.DataKind_int32, false).
		AddField("tpapi_permissions", appdef.DataKind_bytes, false).
		AddField("hide_wm", appdef.DataKind_int32, false).
		AddField("creation_date", appdef.DataKind_int64, false)

	comps := adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "computers"))
	comps.
		AddField("name", appdef.DataKind_string, false).
		AddField("sys.IsActive", appdef.DataKind_bool, false).
		AddField("show_cursor", appdef.DataKind_int32, false).
		AddField("on_hold", appdef.DataKind_int32, false).
		AddField("untillsrv_port", appdef.DataKind_int32, false).
		AddField("id_screen_groups", appdef.DataKind_RecordID, false).
		AddField("id_tickets_clock", appdef.DataKind_RecordID, false).
		AddField("guid_printers_clock", appdef.DataKind_string, false).
		AddField("keyboard_input_text", appdef.DataKind_int32, false).
		AddField("extra_data", appdef.DataKind_string, false).
		AddField("extra_data_new", appdef.DataKind_string, false).
		AddField("startup_message", appdef.DataKind_string, false).
		AddField("guid_cash_printers", appdef.DataKind_string, false).
		AddField("id_cash_tickets", appdef.DataKind_RecordID, false).
		AddField("term_uid", appdef.DataKind_string, false).
		AddField("production_nr", appdef.DataKind_string, false).
		AddField("tpapi", appdef.DataKind_int32, false).
		AddField("ignore_prn_errors", appdef.DataKind_bytes, false).
		AddField("default_a4_printer", appdef.DataKind_string, false).
		AddField("login_screen", appdef.DataKind_int32, false).
		AddField("id_themes", appdef.DataKind_RecordID, false).
		AddField("device_profile_wsid", appdef.DataKind_int64, false)
	comps.
		AddContainer("restaurant_computers", appdef.NewQName(appdef.SysPackage, "restaurant_computers"), appdef.Occurs(0), appdef.Occurs(defaultMaxOccurs))

	adf.AddCRecord(appdef.NewQName(appdef.SysPackage, "restaurant_computers")).
		AddField("id_computers", appdef.DataKind_RecordID, true).
		AddField("id_sales_area", appdef.DataKind_RecordID, false).
		AddField("sales_kind", appdef.DataKind_int32, false).
		AddField("id_printers_1", appdef.DataKind_RecordID, false).
		AddField("keep_waiter", appdef.DataKind_int32, false).
		AddField("limited", appdef.DataKind_int32, false).
		AddField("id_periods", appdef.DataKind_RecordID, false).
		AddField("dbl", appdef.DataKind_int32, false).
		AddField("a4", appdef.DataKind_int32, false).
		AddField("id_screens_part", appdef.DataKind_RecordID, false).
		AddField("id_screens_order", appdef.DataKind_RecordID, false).
		AddField("id_screens_supplement", appdef.DataKind_RecordID, false).
		AddField("id_screens_condiment", appdef.DataKind_RecordID, false).
		AddField("id_screens_payment", appdef.DataKind_RecordID, false).
		AddField("id_tickets_bill", appdef.DataKind_RecordID, false).
		AddField("id_printers_proforma", appdef.DataKind_RecordID, false).
		AddField("id_tickets_proforma", appdef.DataKind_RecordID, false).
		AddField("direct_table", appdef.DataKind_int32, false).
		AddField("start_table", appdef.DataKind_int32, false).
		AddField("id_psp_layout", appdef.DataKind_RecordID, false).
		AddField("id_deposit_layout", appdef.DataKind_RecordID, false).
		AddField("id_deposit_printer", appdef.DataKind_RecordID, false).
		AddField("id_invoice_layout", appdef.DataKind_RecordID, false).
		AddField("id_rear_disp_printer", appdef.DataKind_RecordID, false).
		AddField("id_rear_disp_article_layout", appdef.DataKind_RecordID, false).
		AddField("id_rear_disp_bill_layout", appdef.DataKind_RecordID, false).
		AddField("id_journal_printer", appdef.DataKind_RecordID, false).
		AddField("auto_logoff_sec", appdef.DataKind_int32, false).
		AddField("id_tickets_journal", appdef.DataKind_RecordID, false).
		AddField("future_table", appdef.DataKind_int32, false).
		AddField("id_beco_location", appdef.DataKind_RecordID, false).
		AddField("id_tickets_order_journal", appdef.DataKind_RecordID, false).
		AddField("id_tickets_control_journal", appdef.DataKind_RecordID, false).
		AddField("id_drawer_layout", appdef.DataKind_RecordID, false).
		AddField("table_info", appdef.DataKind_string, false).
		AddField("table_pc_font", appdef.DataKind_bytes, false).
		AddField("table_hht_font", appdef.DataKind_bytes, false).
		AddField("id_return_layout", appdef.DataKind_RecordID, false).
		AddField("id_inout_layout", appdef.DataKind_RecordID, false).
		AddField("id_inout_printer", appdef.DataKind_RecordID, false).
		AddField("id_rent_layout", appdef.DataKind_RecordID, false).
		AddField("id_rent_printer", appdef.DataKind_RecordID, false).
		AddField("id_tickets_preauth", appdef.DataKind_RecordID, false).
		AddField("id_oif_preparation_area", appdef.DataKind_RecordID, false).
		AddField("id_reprint_order", appdef.DataKind_RecordID, false).
		AddField("id_rear_screen_saver", appdef.DataKind_RecordID, false).
		AddField("screen_saver_min", appdef.DataKind_int32, false).
		AddField("notprintlogoff", appdef.DataKind_int32, false).
		AddField("notprintnoorder", appdef.DataKind_int32, false).
		AddField("block_new_client", appdef.DataKind_int32, false).
		AddField("id_init_ks", appdef.DataKind_RecordID, false).
		AddField("id_tickets_giftcards", appdef.DataKind_RecordID, false).
		AddField("id_printers_giftcards", appdef.DataKind_RecordID, false).
		AddField("t2o_prepaid_tablenr", appdef.DataKind_int32, false).
		AddField("t2o_groups_table_from", appdef.DataKind_int32, false).
		AddField("t2o_groups_table_till", appdef.DataKind_int32, false).
		AddField("t2o_clients_table_from", appdef.DataKind_int32, false).
		AddField("t2o_clients_table_till", appdef.DataKind_int32, false).
		AddField("ao_order_direct_sales", appdef.DataKind_int32, false).
		AddField("ao_order_to_table", appdef.DataKind_int32, false).
		AddField("ao_table_nr", appdef.DataKind_int32, false).
		AddField("not_logoff_hht", appdef.DataKind_int32, false).
		AddField("id_printers_voucher", appdef.DataKind_RecordID, false).
		AddField("id_tickets_voucher", appdef.DataKind_RecordID, false).
		AddField("id_email_invoice_layout", appdef.DataKind_RecordID, false).
		AddField("id_printers_manager", appdef.DataKind_RecordID, false).
		AddField("id_tickets_manager", appdef.DataKind_RecordID, false).
		AddField("id_stock_location", appdef.DataKind_RecordID, false).
		AddField("on_hold_printing", appdef.DataKind_int32, false).
		AddField("id_ticket_voucher_bunch", appdef.DataKind_RecordID, false).
		AddField("id_ticket_voucher_bill", appdef.DataKind_RecordID, false).
		AddField("id_stock_printer", appdef.DataKind_RecordID, false).
		AddField("id_coupon_layout", appdef.DataKind_RecordID, false).
		AddField("id_printers_taorder", appdef.DataKind_RecordID, false).
		AddField("id_tickets_taorder", appdef.DataKind_RecordID, false).
		AddField("id_second_article_layout", appdef.DataKind_RecordID, false).
		AddField("second_article_delay_sec", appdef.DataKind_int32, false).
		AddField("id_printers_void", appdef.DataKind_RecordID, false).
		AddField("id_tickets_void", appdef.DataKind_RecordID, false).
		AddField("id_tickets_fiscal_footer", appdef.DataKind_RecordID, false).
		AddField("temp_orders_table_from", appdef.DataKind_int32, false).
		AddField("temp_orders_table_to", appdef.DataKind_int32, false).
		AddField("id_init_ksc", appdef.DataKind_RecordID, false).
		AddField("use_word_template_print_invoice", appdef.DataKind_int32, false).
		AddField("id_ta_total_layout", appdef.DataKind_RecordID, false).
		AddField("notify_blocked_card", appdef.DataKind_int32, false).
		AddField("id_printers_reopen", appdef.DataKind_RecordID, false).
		AddField("id_tickets_reopen", appdef.DataKind_RecordID, false).
		AddField("notify_blocked_card_layer", appdef.DataKind_int32, false).
		AddField("id_tickets_prof_fiscal_footer", appdef.DataKind_RecordID, false).
		AddField("id_tickets_giftcardsbill", appdef.DataKind_RecordID, false).
		AddField("id_printers_giftcardsbill", appdef.DataKind_RecordID, false)

	adf.AddWDoc(appdef.NewQName(appdef.SysPackage, "bill")).
		AddField("tableno", appdef.DataKind_int32, true).
		AddField("id_untill_users", appdef.DataKind_RecordID, true).
		AddField("table_part", appdef.DataKind_string, true).
		AddField("id_courses", appdef.DataKind_RecordID, false).
		AddField("id_clients", appdef.DataKind_RecordID, false).
		AddField("name", appdef.DataKind_string, false).
		AddField("proforma", appdef.DataKind_int32, true).
		AddField("modified", appdef.DataKind_int64, false).
		AddField("open_datetime", appdef.DataKind_int64, false).
		AddField("close_datetime", appdef.DataKind_int64, false).
		AddField("number", appdef.DataKind_int32, false).
		AddField("failurednumber", appdef.DataKind_int32, false).
		AddField("suffix", appdef.DataKind_string, false).
		AddField("pbill_number", appdef.DataKind_int32, false).
		AddField("pbill_failurednumber", appdef.DataKind_int32, false).
		AddField("pbill_suffix", appdef.DataKind_string, false).
		AddField("hc_foliosequence", appdef.DataKind_int32, false).
		AddField("hc_folionumber", appdef.DataKind_string, false).
		AddField("tip", appdef.DataKind_int64, false).
		AddField("qty_persons", appdef.DataKind_int32, false).
		AddField("isdirty", appdef.DataKind_int32, false).
		AddField("reservationid", appdef.DataKind_string, false).
		AddField("id_alter_user", appdef.DataKind_RecordID, false).
		AddField("service_charge", appdef.DataKind_float64, false).
		AddField("number_of_covers", appdef.DataKind_int32, false).
		AddField("id_user_proforma", appdef.DataKind_RecordID, false).
		AddField("bill_type", appdef.DataKind_int32, false).
		AddField("locker", appdef.DataKind_int32, false).
		AddField("id_time_article", appdef.DataKind_RecordID, false).
		AddField("timer_start", appdef.DataKind_int64, false).
		AddField("timer_stop", appdef.DataKind_int64, false).
		AddField("isactive", appdef.DataKind_int32, false).
		AddField("table_name", appdef.DataKind_string, false).
		AddField("group_vat_level", appdef.DataKind_int32, false).
		AddField("comments", appdef.DataKind_string, false).
		AddField("id_cardprice", appdef.DataKind_RecordID, false).
		AddField("discount", appdef.DataKind_float64, false).
		AddField("discount_value", appdef.DataKind_int64, false).
		AddField("id_discount_reasons", appdef.DataKind_RecordID, false).
		AddField("hc_roomnumber", appdef.DataKind_string, false).
		AddField("ignore_auto_sc", appdef.DataKind_int32, false).
		AddField("extra_fields", appdef.DataKind_bytes, false).
		AddField("id_bo_service_charge", appdef.DataKind_RecordID, false).
		AddField("free_comments", appdef.DataKind_string, false).
		AddField("id_t2o_groups", appdef.DataKind_RecordID, false).
		AddField("service_tax", appdef.DataKind_int64, false).
		AddField("sc_plan", appdef.DataKind_bytes, false).
		AddField("client_phone", appdef.DataKind_string, false).
		AddField("age", appdef.DataKind_int64, false).
		AddField("description", appdef.DataKind_bytes, false).
		AddField("sdescription", appdef.DataKind_string, false).
		AddField("vars", appdef.DataKind_bytes, false).
		AddField("take_away", appdef.DataKind_int32, false).
		AddField("fiscal_number", appdef.DataKind_int32, false).
		AddField("fiscal_failurednumber", appdef.DataKind_int32, false).
		AddField("fiscal_suffix", appdef.DataKind_string, false).
		AddField("id_order_type", appdef.DataKind_RecordID, false).
		AddField("not_paid", appdef.DataKind_int64, false).
		AddField("total", appdef.DataKind_int64, false).
		AddField("was_cancelled", appdef.DataKind_int32, false).
		AddField("id_callers_last", appdef.DataKind_RecordID, false).
		AddField("id_serving_time", appdef.DataKind_RecordID, false).
		AddField("serving_time_dt", appdef.DataKind_int64, false).
		AddField("vat_excluded", appdef.DataKind_int32, false).
		AddField("day_number", appdef.DataKind_int32, false).
		AddField("day_failurednumber", appdef.DataKind_int32, false).
		AddField("day_suffix", appdef.DataKind_string, false).
		AddField("ayce_time", appdef.DataKind_int64, false).
		AddField("remaining_quantity", appdef.DataKind_float64, false).
		AddField("working_day", appdef.DataKind_string, true)

	adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "pos_emails")).
		AddField("kind", appdef.DataKind_int32, false).
		AddField("email", appdef.DataKind_string, false).
		AddField("description", appdef.DataKind_string, false).
		AddField("sys.IsActive", appdef.DataKind_bool, false)

	adf.AddSingleton(QNameTestWSKind).
		AddField("IntFld", appdef.DataKind_int32, true).
		AddField("StrFld", appdef.DataKind_string, false)

	adf.AddCDoc(appdef.NewQName(appdef.SysPackage, "category")).
		AddField("name", appdef.DataKind_string, false).
		AddField("sys.IsActive", appdef.DataKind_bool, false).
		AddField("hq_id", appdef.DataKind_string, false).
		AddField("ml_name", appdef.DataKind_bytes, false).
		AddField("cat_external_id", appdef.DataKind_string, false)

	projectors.ProvideViewDef(adf, QNameTestView, func(b appdef.IViewBuilder) {
		b.AddPartField("ViewIntFld", appdef.DataKind_int32).
			AddClustColumn("ViewStrFld", appdef.DataKind_string)
	})
	sep.ExtensionPoint(wskinds.EPWorkspaceKind).Add(QNameTestWSKind)

	// for impl_verifier_test
	adf.AddCDoc(QNameTestEmailVerificationDoc).
		AddVerifiedField("EmailField", appdef.DataKind_string, true, appdef.VerificationKind_EMail).
		AddVerifiedField("PhoneField", appdef.DataKind_string, false, appdef.VerificationKind_Phone).
		AddField("NonVerifiedField", appdef.DataKind_string, false)

	// for impl_uniques_test
	doc := adf.AddCDoc(QNameCDocTestConstraints)
	doc.AddField("Int", appdef.DataKind_int32, true).
		AddField("Str", appdef.DataKind_string, true).
		AddField("Bool", appdef.DataKind_bool, true).
		AddField("Float32", appdef.DataKind_float32, false).
		AddField("Bytes", appdef.DataKind_bytes, true)
	doc.SetUniqueField("Int")

	// for singletons test
	adf.AddSingleton(QNameTestSingleton).
		AddField("Fld1", appdef.DataKind_string, true)

	// for rates test
	pars := adf.AddObject(appdef.NewQName(appdef.SysPackage, "RatedQryParams"))
	pars.AddField("Fld", appdef.DataKind_string, false)
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQryRated, appdef.NullQName, pars.QName(),
		istructsmem.NullQueryExec,
	))
	pars = adf.AddObject(appdef.NewQName(appdef.SysPackage, "RatedCmdParams"))
	pars.AddField("Fld", appdef.DataKind_string, false)
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCmdRated, pars.QName(), appdef.NullQName, appdef.NullQName,
		istructsmem.NullCommandExec,
	))

	// per-app limits
	cfg.FunctionRateLimits.AddAppLimit(QNameCmdRated, istructs.RateLimit{
		Period:                time.Minute,
		MaxAllowedPerDuration: 2,
	})
	cfg.FunctionRateLimits.AddAppLimit(QNameQryRated, istructs.RateLimit{
		Period:                time.Minute,
		MaxAllowedPerDuration: 2,
	})

	// per-workspace limits
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameCmdRated, istructs.RateLimit{
		Period:                time.Hour,
		MaxAllowedPerDuration: 4,
	})
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameQryRated, istructs.RateLimit{
		Period:                time.Hour,
		MaxAllowedPerDuration: 4,
	})

	mockQryQName := appdef.NewQName(appdef.SysPackage, "MockQry")
	mockQryParamsQName := appdef.NewQName(appdef.SysPackage, "MockQryParams")
	adf.AddObject(mockQryParamsQName).
		AddField(field_Input, appdef.DataKind_string, true)

	mockQryResQName := appdef.NewQName(appdef.SysPackage, "MockQryResult")
	mockQryResScheme := adf.AddObject(mockQryResQName)
	mockQryResScheme.AddField("Res", appdef.DataKind_string, true)

	mockQry := istructsmem.NewQueryFunction(mockQryQName, mockQryParamsQName, mockQryResQName,
		func(_ context.Context, _ istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
			input := args.ArgumentObject.AsString(field_Input)
			return MockQryExec(input, callback)
		},
	)
	cfg.Resources.Add(mockQry)

	mockCmdQName := appdef.NewQName(appdef.SysPackage, "MockCmd")
	mockCmdParamsQName := appdef.NewQName(appdef.SysPackage, "MockCmdParams")
	adf.AddObject(mockCmdParamsQName).
		AddField(field_Input, appdef.DataKind_string, true)

	execCmdMockCmd := func(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
		input := args.ArgumentObject.AsString(field_Input)
		return MockCmdExec(input)
	}
	mockCmd := istructsmem.NewCommandFunction(mockCmdQName, mockCmdParamsQName, appdef.NullQName, appdef.NullQName, execCmdMockCmd)
	cfg.Resources.Add(mockCmd)
}
