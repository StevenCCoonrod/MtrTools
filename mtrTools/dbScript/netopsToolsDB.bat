echo off

rem For Use at home
rem sqlcmd -S localhost\MSSQLSERVER02 -E -i NetopsToolsDB.sql

rem Generic Local Host
sqlcmd -S localhost -E -i NetopsToolsDB.sql

rem Generic SqlServer
rem sqlcmd -S localhost\mssqlserver -E -i NetopsToolsDB.sql

rem sqlcmd -S ec2-54-204-1-34.compute-1.amazonaws.com -E -i NetopsToolsDB.sql

echo .
echo if no error messages appear, sample data was added to the database
pause