echo off

rem For Use at home
sqlcmd -S localhost\MSSQLSERVER02 -E -i NetopsToolsDB.sql
echo .
echo if no error messages appear DB was created 
pause

rem Generic Local Host
rem sqlcmd -S localhost\ -E -i NetopsToolsDB.sql
rem echo .
rem echo if no error messages appear, database was created successfully
rem pause

rem Generic SqlServer
rem sqlcmd -S localhost\mssqlserver -E -i NetopsToolsDB.sql
rem echo .
rem echo if no error messages appear, sample data was added to the database
