CREATE DATABASE /*! IF NOT EXISTS*/`NetopsToolsDB`;

USE `NetopsToolsDB`;

CREATE TABLE `Syncbox`
(
	`SyncboxID`		VARCHAR(15)		NOT NULL,
	
	CONSTRAINT `pk_SyncboxID` PRIMARY KEY (`SyncboxID`)
);

CREATE TABLE `MtrReport`
(
	`MtrReportID`		INT				NOT NULL AUTO_INCREMENT,
	`SyncboxID`			VARCHAR(15)		NOT NULL,
	`StartTime`			DATETIME		NOT NULL,
	`DataCenter`		VARCHAR(2),
	
	CONSTRAINT `fk_SyncboxID` FOREIGN KEY (`SyncboxID`)  REFERENCES `dbo`.`Syncbox`(`SyncboxID`),
	CONSTRAINT `pk_MtrReportID` PRIMARY KEY (`MtrReportID`)
);

CREATE TABLE `MtrHop`
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

	CONSTRAINT `fk_mtrReport` FOREIGN KEY (`MtrReportID`)  REFERENCES `dbo`.`MtrReport`(`MtrReportID`),
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

DELIMITER //
CREATE PROCEDURE sp_InsertSyncbox
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

DELIMITER //
CREATE PROCEDURE sp_InsertMtrReport
(
	IN syncboxID			VARCHAR(15),
	IN startTime			DATETIME,
	IN dataCenter			VARCHAR(2)
)
IF NOT EXISTS (SELECT SyncboxID FROM Syncbox WHERE SyncboxID=syncboxID)
	THEN CALL sp_InsertSyncbox(syncboxID);
END IF;
IF NOT EXISTS (SELECT MtrReportID FROM MtrReport WHERE SyncboxID=syncboxID AND StartTime=startTime AND DataCenter=dataCenter)
	THEN BEGIN
		INSERT INTO MtrReport
			(SyncboxID,StartTime,DataCenter)
		VALUES
			(UPPER(syncboxID), startTime, dataCenter);
		SELECT LAST_INSERT_ID();
	END;
END IF; 
END//
DELIMITER ;

DELIMITER //
CREATE PROCEDURE sp_InsertMtrHop
(
	IN mtrReportID		VARCHAR(50), 
	IN hopNumber		TINYINT,
	IN hostName			VARCHAR(200),
	IN packetLoss		DECIMAL,
	IN packetsSent		TINYINT,
	IN lastPingMS		DECIMAL,
	IN avgPingMS		DECIMAL,
	IN bestPingMS		DECIMAL,
	IN worstPingMS		DECIMAL,
	IN standardDev		DECIMAL
)
BEGIN
	INSERT INTO MtrHop
		(MtrReportID, HopNumber, HostName,PacketLoss,PacketsSent,LastPingMS,AvgPingMS,BestPingMS,WorstPingMS,StandardDev)
	VALUES
		(mtrReportID, hopNumber, hostName, packetLoss, packetsSent, lastPingMS, avgPingMS, bestPingMS, worstPingMS, standardDev);
	SELECT LAST_INSERT_ID();
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

DELIMITER //
CREATE PROCEDURE sp_SelectAllSyncboxes()
BEGIN
	SELECT 	SyncboxID
	FROM 	Syncbox;
END//
DELIMITER ;

DELIMITER //
CREATE PROCEDURE sp_SelectMtrReportByID
(
	IN mtrReportID		VARCHAR(50)
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
	FROM	MtrReport	INNER JOIN MtrHop
		ON	MtrReport.MtrReportID = MtrHop.MtrReportID
		WHERE MtrReport.MtrReportID = mtrReportID;
END//
DELIMITER ;

DELIMITER //
CREATE PROCEDURE sp_CheckIfMtrReportExists
(
	IN mtrReportID		VARCHAR(50)
)
BEGIN
	SELECT 	MtrReport.MtrReportID
	FROM	MtrReport
	WHERE 	MtrReport.MtrReportID = mtrReportID;
END//
DELIMITER ;

DELIMITER //
CREATE PROCEDURE sp_SelectAllMtrs()
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
	ORDER BY MtrReportID;
END//
DELIMITER ;

DELIMITER //
CREATE PROCEDURE sp_SelectAllMtrsWithinRange
(
	IN startDatetime		DATETIME,
	IN endDatetime			DATETIME
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
	WHERE 	MtrReport.StartTime >= startDatetime
	AND		MtrReport.StartTime <= endDatetime
	ORDER BY SyncboxID, MtrReportID;
END//
DELIMITER ;

DELIMITER //
CREATE PROCEDURE sp_SelectMtrs_BySyncbox_WithinRange
(
	IN syncboxID			VARCHAR(15),
	IN startDatetime		DATETIME,
	IN endDatetime			DATETIME
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
	WHERE 	SyncboxID = syncboxID
	AND		MtrReport.StartTime >= startDatetime
	AND		MtrReport.StartTime <= endDatetime
	ORDER BY MtrReportID;
END//
DELIMITER ;

DELIMITER //
CREATE PROCEDURE sp_SelectMtrs_BySyncbox_DCAndTimeframe
(
	IN syncboxID			VARCHAR(15),
	IN startDatetime		DATETIME,
	IN endDatetime			DATETIME,
	IN dataCenter			VARCHAR(2)
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
	FROM	MtrReport
			INNER JOIN MtrHop
		ON	MtrReport.MtrReportID = MtrHop.MtrReportID
	WHERE 	SyncboxID = syncboxID
	AND		MtrReport.StartTime >=  startDatetime
	AND		MtrReport.StartTime <=  endDatetime
	AND 	MtrReport.DataCenter =  dataCenter
	ORDER BY MtrReportID, MtrHopID;
END//
DELIMITER ;

DELIMITER //
CREATE PROCEDURE sp_SelectMtrs_ByHostname
(
	IN 	hostname		VARCHAR(200)
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
	WHERE 	HostName = hostname
	ORDER BY MtrReportID;
END//
DELIMITER ;






