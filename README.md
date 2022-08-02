# MTR Tools

Golang project for storing and retrieving MTR Data.<br />
Retrieves traceroute data via SSH and inserts data into a Database.<br />
Houses tools for analyzing traceroute data.

## Flags:<br />
**-a**        &emsp;&emsp;&emsp;&emsp;&emsp;&nbsp;Will run a sweep of ALL mtr data for ALL Syncboxes. <br />
**-start**    &emsp;&emsp;&emsp;&nbsp;&nbsp;&nbsp;Specify a target start time. eg. 5h30m = 5hours and 30 minutes in the past <br />
**-end**      &emsp;&emsp;&emsp;&emsp;&nbsp;Specify a target end time. eg. 0m = now, 0 minutes in the past<br />
**-p**        &emsp;&emsp;&emsp;&emsp;&emsp;&nbsp;Print results to command-line <br />
**syncboxID** &emsp;&ensp;&nbsp;Will run a sweep of All mtr for the specified Syncbox. <br />
**no args**   &emsp;&emsp;&emsp;Will run a test method.
  
## Actions for set up:<br />
  &emsp;Requires SQL Server <br />
  &emsp;Run the dbScript/netopsToolsDB batch file to establish the DB.<br />
  &emsp;Enter your user id and password in the sshHelpers.go and the sqlDataAccessor.go files.
