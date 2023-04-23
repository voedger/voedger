SET curl=E:\curl-7.62.0-win64-mingw\bin\curl.exe
%curl% -X PUT -d @template_auditlog.json http://127.0.0.1:9200/_template/auditlog --header "Content-Type: application/json" --basic --user admin:admin
