SET curl=E:\curl-7.62.0-win64-mingw\bin\curl.exe
%curl% -X PUT -d @index_settings.json http://127.0.0.1:9200/security-auditlog-2023.02.01/_settings --header "Content-Type: application/json" --basic --user admin:admin
