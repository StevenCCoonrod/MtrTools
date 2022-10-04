CREATE DATABASE /*! IF NOT EXISTS*/`NetopsToolsDB`;

USE `NetopsToolsDB`;
DROP TABLE IF EXISTS `NetopsToolsDB`.`MtrHop`;
DROP TABLE IF EXISTS `NetopsToolsDB`.`MtrReport`;
DROP TABLE IF EXISTS `NetopsToolsDB`.`Syncbox`;
CREATE TABLE `NetopsToolsDB`.`Syncbox`
(
	`SyncboxID`		VARCHAR(15)		NOT NULL,
	
	CONSTRAINT `pk_SyncboxID` PRIMARY KEY (`SyncboxID`)
);

CREATE TABLE `NetopsToolsDB`.`MtrReport`
(
	`MtrReportID`		INT				NOT NULL AUTO_INCREMENT,
	`SyncboxID`			VARCHAR(15)		NOT NULL,
	`StartTime`			DATETIME		NOT NULL,
	`DataCenter`		VARCHAR(2),
	
	CONSTRAINT `fk_SyncboxID` FOREIGN KEY (`SyncboxID`)  REFERENCES `Syncbox`(`SyncboxID`),
	CONSTRAINT `pk_MtrReportID` PRIMARY KEY (`MtrReportID`)
);

CREATE TABLE `NetopsToolsDB`.`MtrHop`
(
	`MtrHopID`			INT				NOT NULL	AUTO_INCREMENT,
	`MtrReportID`		INT				NOT NULL,
	`HopNumber`			TINYINT			NOT NULL,
	`HostName`			VARCHAR(200) 	NOT NULL,
	`PacketLoss`		DECIMAL			NOT NULL,
	`PacketsSent`		TINYINT			NOT NULL,
	`LastPingMS`		DECIMAL			NOT NULL,
	`AvgPingMS`			DECIMAL			NOT NULL,
	`BestPingMS`		DECIMAL			NOT NULL,
	`WorstPingMS`		DECIMAL			NOT NULL,
	`StandardDev`		DECIMAL			NOT NULL,	

	CONSTRAINT `fk_mtrReport` FOREIGN KEY (`MtrReportID`)  REFERENCES `MtrReport`(`MtrReportID`),
	CONSTRAINT `pk_mtrHopID` PRIMARY KEY (`MtrHopID`)
);


/*-------------------------------------------------------------------------------------------------------------------------------------------*/
/*|||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||*/
/*-------------------------------------------------------------------------------------------------------------------------------------------*/

/*															STORED PROCEDURES

/*-------------------------------------------------------------------------------------------------------------------------------------------*/
/*|||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||*/
/*-------------------------------------------------------------------------------------------------------------------------------------------*/
/*																INSERT 
/*-------------------------------------------------------------------------------------------------------------------------------------------*/
/*|||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||*/
/*-------------------------------------------------------------------------------------------------------------------------------------------*/
DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_InsertSyncbox`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_InsertSyncbox`
(
	IN 	syncboxID		VARCHAR(15)
)
BEGIN
	INSERT INTO Syncbox
		(SyncboxID)
	VALUES
		(UPPER(syncboxID));
END //
DELIMITER ;
DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_InsertMtrReport`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_InsertMtrReport`
(
	IN 		syncboxID			VARCHAR(15),
	IN 		startTime			DATETIME,
	IN 		dataCenter			VARCHAR(2)
)
BEGIN
	DECLARE reportID INT DEFAULT 0;
	IF NOT EXISTS 
		(SELECT NetopsToolsDB.MtrReport.MtrReportID 
		 FROM 	NetopsToolsDB.MtrReport 
         WHERE 	NetopsToolsDB.MtrReport.SyncboxID = syncboxID 
			AND NetopsToolsDB.MtrReport.StartTime = startTime 
            AND NetopsToolsDB.MtrReport.DataCenter = dataCenter)
	THEN 
		INSERT INTO NetopsToolsDB.MtrReport
			(SyncboxID,StartTime,DataCenter)
		VALUES
			(UPPER(syncboxID), startTime, dataCenter);
		SELECT LAST_INSERT_ID() INTO reportID;
	END IF;
    SELECT reportID;
END//
DELIMITER ;
DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_InsertMtrHop`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_InsertMtrHop`
(
	IN 		mtrReportID		VARCHAR(50), 
	IN 		hopNumber		TINYINT,
	IN 		hostName		VARCHAR(200),
	IN 		packetLoss		DECIMAL,
	IN 		packetsSent		TINYINT,
	IN 		lastPingMS		DECIMAL,
	IN 		avgPingMS		DECIMAL,
	IN 		bestPingMS		DECIMAL,
	IN 		worstPingMS		DECIMAL,
	IN 		standardDev		DECIMAL
)
BEGIN
	DECLARE 	hopID	INT DEFAULT 0;
	INSERT INTO MtrHop
		(MtrReportID, HopNumber, HostName,PacketLoss,PacketsSent,LastPingMS,AvgPingMS,BestPingMS,WorstPingMS,StandardDev)
	VALUES
		(mtrReportID, hopNumber, hostName, packetLoss, packetsSent, lastPingMS, avgPingMS, bestPingMS, worstPingMS, standardDev);
	SELECT LAST_INSERT_ID() INTO hopID;
END//
DELIMITER ;

/*
---------------------------------------------------------------------------------------------------------------------------------------------
--|||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
---------------------------------------------------------------------------------------------------------------------------------------------
--															SELECT Statements
---------------------------------------------------------------------------------------------------------------------------------------------
--|||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
---------------------------------------------------------------------------------------------------------------------------------------------
*/
DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_CheckIfMtrReportExists`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_CheckIfMtrReportExists`
(
	IN 		syncbox_ID			VARCHAR(15),
	IN 		start_Time			DATETIME,
	IN 		data_Center			VARCHAR(2)
)
BEGIN
	SELECT 	MtrReport.MtrReportID
	FROM	MtrReport
	WHERE 	MtrReport.SyncboxID = UPPER(syncbox_ID)
		AND	StartTime = start_Time
        AND DataCenter = data_Center;
END//
DELIMITER ;


DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_SelectAllSyncboxes`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_SelectAllSyncboxes`()
BEGIN
	SELECT 	SyncboxID
	FROM 	Syncbox
	ORDER BY SyncboxID;
END//
DELIMITER ;

DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_SelectAllMtrs_BySyncbox`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_SelectAllMtrs_BySyncbox`(
	IN 	syncbox_id		VARCHAR(15)
)
BEGIN
	SELECT 	MtrReportID,
			SyncboxID,
			StartTime,
			DataCenter
	FROM	NetopsToolsDB.MtrReport 
	WHERE 	SyncboxID = UPPER(syncbox_id)
	ORDER BY StartTime, DataCenter;
END//
DELIMITER ;


DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_SelectAllMtrs`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_SelectAllMtrs`()
BEGIN
	SELECT 	NetopsToolsDB.MtrReport.MtrReportID,
			NetopsToolsDB.MtrReport.SyncboxID,
			NetopsToolsDB.MtrReport.StartTime,
			NetopsToolsDB.MtrReport.DataCenter
	FROM	NetopsToolsDB.MtrReport 
	ORDER BY SyncboxID, StartTime, DataCenter;
END//
DELIMITER ;


DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_SelectAllMtrs_WithinRange`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_SelectAllMtrs_WithinRange`
(
	IN start_Datetime		DATETIME,
	IN end_Datetime			DATETIME
)
BEGIN
	SELECT 	MtrReport.MtrReportID,
			SyncboxID,
			StartTime,
			DataCenter
	FROM	NetopsToolsDB.MtrReport
	WHERE 	MtrReport.StartTime >= start_Datetime
	AND		MtrReport.StartTime <= end_Datetime
	ORDER BY SyncboxID, StartTime, HopNumber;
END//
DELIMITER ;


DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_SelectAllMtrs_BySyncbox_WithinRange`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_SelectAllMtrs_BySyncbox_WithinRange`
(
	IN syncbox_ID			VARCHAR(15),
	IN start_Datetime		DATETIME,
	IN end_Datetime			DATETIME
)
BEGIN
	SELECT 	MtrReport.MtrReportID,
			SyncboxID,
			StartTime,
			DataCenter
	FROM	NetopsToolsDB.MtrReport
	WHERE 	MtrReport.SyncboxID = syncbox_ID
	AND		MtrReport.StartTime >= start_Datetime
	AND		MtrReport.StartTime <= end_Datetime
	ORDER BY StartTime;
END//
DELIMITER ;


DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_SelectAllMtrs_BySyncbox_WithinRange_ByDC`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_SelectAllMtrs_BySyncbox_WithinRange_ByDC`
(
	IN syncbox_ID			VARCHAR(15),
	IN start_Datetime		DATETIME,
	IN end_Datetime			DATETIME,
	IN data_Center			VARCHAR(2)
)
BEGIN
	SELECT 	MtrReport.MtrReportID,
			SyncboxID,
			StartTime,
			DataCenter
	FROM	MtrReport
	WHERE 	MtrReport.SyncboxID = syncbox_ID
	AND		MtrReport.StartTime >=  start_Datetime
	AND		MtrReport.StartTime <=  end_Datetime
	AND 	MtrReport.DataCenter =  data_Center
	ORDER BY StartTime;
END//
DELIMITER ;


DROP PROCEDURE IF EXISTS `NetopsToolsDB`.`sp_SelectAllMtrs_ByHostname`;
DELIMITER //
CREATE DEFINER=`strmdashdb`@`%` PROCEDURE `NetopsToolsDB`.`sp_SelectAllMtrs_ByHostname`
(
	IN 	_hostname		VARCHAR(200)
)
BEGIN
	SELECT 	MtrReport.MtrReportID,
			SyncboxID,
			StartTime,
			DataCenter,
			MtrHopID,
			HopNumber,
			HostName,
			PacketLoss,
			PacketsSent,
			LastPingMS,
			AvgPingMS,
			BestPingMS,
			WorstPingMS,
			StandardDev
	FROM	MtrReport INNER JOIN MtrHop
		ON	MtrReport.MtrReportID = MtrHop.MtrReportID
	WHERE 	HostName = _hostname
	ORDER BY SyncboxID, StartTime, HopNumber;
END//
DELIMITER ;





