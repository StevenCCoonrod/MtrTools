# NetopsTools

Golang project for storing and retrieving MTR Data.<br />
Retrieves traceroute data via SSH and inserts data into a Database.<br />
Houses tools for analyzing traceroute data.

## Flags:<br />
**-a**        &emsp;&emsp;&emsp;&emsp;&emsp;&nbsp;Will run a sweep of ALL mtr data for ALL Syncboxes. <br />
**-start**    &emsp;&emsp;&emsp;&nbsp;&nbsp;&nbsp;Specify a target start time <br />
**-end**      &emsp;&emsp;&emsp;&emsp;&nbsp;Specify a target end time <br />
**-p**        &emsp;&emsp;&emsp;&emsp;&emsp;&nbsp;Print results to command-line <br />
**syncboxID** &emsp;&ensp;&nbsp;Will run a sweep of All mtr for the specified Syncbox. <br />
**no args**   &emsp;&emsp;&emsp;Will run a test method.
  
## Actions for set up:<br />
  &emsp;Requires SQL Server
  &emsp;Run the DB script batch file to establish the DB.<br />
  &emsp;Enter your user id and password for both the sshHelpers and the sqlDataAccessor.
