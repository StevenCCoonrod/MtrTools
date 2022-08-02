IF EXISTS (SELECT 1 FROM master.dbo.sysdatabases WHERE name = 'NetopsToolsDB')
	BEGIN
		DROP DATABASE [NetopsToolsDB]
		print '' print '> Dropping NetopsToolsDB'
	END
GO
print '' print '> Creating NetopsToolsDB'
GO
CREATE DATABASE [NetopsToolsDB]
GO
print '' print '> Using NetopsToolsDB'
GO
USE [NetopsToolsDB]
GO

CREATE TABLE [dbo].[Syncbox]
(
	[SyncboxID]		VARCHAR(15)		NOT NULL
	
	CONSTRAINT [pk_SyncboxID] PRIMARY KEY ([SyncboxID])
);
GO

print '' print '> Syncbox table created'
GO

CREATE TABLE [dbo].[MtrReport]
(
	[MtrReportID]		VARCHAR(50)		NOT NULL,
	[SyncboxID]			VARCHAR(15)		NOT NULL,
	[StartTime]			DATETIME		NOT NULL,
	[DataCenter]		VARCHAR(2),
	
	CONSTRAINT [fk_SyncboxID] FOREIGN KEY ([SyncboxID])  REFERENCES [dbo].[Syncbox]([SyncboxID]),
	CONSTRAINT [pk_MtrReportID] PRIMARY KEY ([MtrReportID])
);
GO

print '' print '> MtrReport table created'
GO

CREATE TABLE [dbo].[MtrHop]
(
	[MtrHopID]			INT				NOT NULL	IDENTITY(100000000,1),
	[MtrReportID]		VARCHAR(50)	NOT NULL,
	[HopNumber]			TINYINT			NOT NULL,
	[HostName]			VARCHAR(200) 	NOT NULL,
	[PacketLoss]		DECIMAL			NOT NULL,
	[PacketsSent]		TINYINT			NOT NULL,
	[LastPingMS]		DECIMAL			NOT NULL,
	[AvgPingMS]			DECIMAL			NOT NULL,
	[BestPingMS]		DECIMAL			NOT NULL,
	[WorstPingMS]		DECIMAL			NOT NULL,
	[StandardDev]		DECIMAL			NOT NULL,	

	CONSTRAINT [fk_mtrReport] FOREIGN KEY ([MtrReportID])  REFERENCES [dbo].[MtrReport]([MtrReportID]),
	CONSTRAINT [pk_mtrHopID] PRIMARY KEY ([MtrHopID])
);
GO

print '' print '> MtrHop table created'
GO



--===============STORED PROCEDURES=================--

--INSERT Statements for SSH MTR Data--


CREATE PROCEDURE [sp_InsertSyncbox]
(
	@SyncboxID		[VARCHAR](15)
)
AS
BEGIN
	INSERT INTO [dbo].[Syncbox]
		([SyncboxID])
	VALUES
		(UPPER(@SyncboxID))
END
GO

CREATE PROCEDURE [sp_InsertMtrReport]
(
	@MtrReportID		[VARCHAR](50),
	@SyncboxID			[VARCHAR](15),
	@StartTime			[DATETIME],
	@DataCenter			[VARCHAR](2)
)
AS
BEGIN
	INSERT INTO [dbo].[MtrReport]
		([MtrReportID],[SyncboxID],[StartTime],[DataCenter])
	VALUES
		(@MtrReportID, UPPER(@SyncboxID), @StartTime, @DataCenter)
END
GO

CREATE PROCEDURE [sp_InsertMtrHop]
(
	@MtrReportID		[VARCHAR](50), 
	@HopNumber			TINYINT,
	@HostName			VARCHAR(200),
	@PacketLoss			DECIMAL,
	@PacketsSent		TINYINT,
	@LastPingMS			DECIMAL,
	@AvgPingMS			DECIMAL,
	@BestPingMS			DECIMAL,
	@WorstPingMS		DECIMAL,
	@StandardDev		DECIMAL
)
AS
BEGIN
	INSERT INTO [dbo].[MtrHop]
		([MtrReportID], [HopNumber], [HostName],[PacketLoss],[PacketsSent],[LastPingMS],[AvgPingMS],[BestPingMS],[WorstPingMS],[StandardDev])
	VALUES
		(@MtrReportID, @HopNumber, @HostName, @PacketLoss, @PacketsSent, @LastPingMS, @AvgPingMS, @BestPingMS, @WorstPingMS, @StandardDev)
	SELECT SCOPE_IDENTITY()
END
GO

--SELECT Statements

CREATE PROCEDURE [sp_SelectAllSyncboxes]
AS
BEGIN
	SELECT 	[SyncboxID]
	FROM 	[dbo].[Syncbox]
END
GO

CREATE PROCEDURE [sp_SelectMtrReportByID]
(
	@MtrReportID		VARCHAR(50)
)
AS
BEGIN
	SELECT 	[dbo].[MtrReport].[MtrReportID],
			[SyncboxID],
			[StartTime],
			[DataCenter],
			[HopNumber],
			[HostName],
			[PacketLoss],
			[PacketsSent],
			[LastPingMS],
			[AvgPingMS],
			[BestPingMS],
			[WorstPingMS],
			[StandardDev]
	FROM	[dbo].[MtrReport]
			INNER JOIN [dbo].[MtrHop]
		ON	[dbo].[MtrReport].[MtrReportID] = [dbo].[MtrHop].[MtrReportID]
		WHERE [dbo].[MtrReport].[MtrReportID] = @MtrReportID
END
GO

CREATE PROCEDURE [sp_CheckIfMtrReportExists]
(
	@MtrReportID		VARCHAR(50)
)
AS
BEGIN
	SELECT 	[dbo].[MtrReport].[MtrReportID]
	FROM	[dbo].[MtrReport]
	WHERE 	[dbo].[MtrReport].[MtrReportID] = @MtrReportID
END
GO


CREATE PROCEDURE [sp_SelectAllMtrs]
AS
BEGIN
	SELECT 	[dbo].[MtrReport].[MtrReportID],
			[SyncboxID],
			[StartTime],
			[DataCenter],
			[HopNumber],
			[HostName],
			[PacketLoss],
			[PacketsSent],
			[LastPingMS],
			[AvgPingMS],
			[BestPingMS],
			[WorstPingMS],
			[StandardDev]
	FROM	[dbo].[MtrReport]
			INNER JOIN [dbo].[MtrHop]
		ON	[dbo].[MtrReport].[MtrReportID] = [dbo].[MtrHop].[MtrReportID]
	ORDER BY [MtrReportID]
END
GO


CREATE PROCEDURE [sp_SelectAllMtrsWithinRange]
(
	@StartDatetime		[DATETIME],
	@EndDatetime		[DATETIME]
)
AS
BEGIN
	SELECT 	[dbo].[MtrReport].[MtrReportID],
			[SyncboxID],
			[StartTime],
			[DataCenter],
			[HopNumber],
			[HostName],
			[PacketLoss],
			[PacketsSent],
			[LastPingMS],
			[AvgPingMS],
			[BestPingMS],
			[WorstPingMS],
			[StandardDev]
	FROM	[dbo].[MtrReport]
			INNER JOIN [dbo].[MtrHop]
		ON	[dbo].[MtrReport].[MtrReportID] = [dbo].[MtrHop].[MtrReportID]
	WHERE 	[MtrReport].[StartTime] >= @StartDatetime
	AND		[MtrReport].[StartTime] <= @EndDatetime
	ORDER BY [SyncboxID], [MtrReportID]
END
GO

CREATE PROCEDURE [sp_SelectSyncboxMtrsWithinRange]
(
	@SyncboxID			[VARCHAR](15),
	@StartDatetime		[DATETIME],
	@EndDatetime		[DATETIME]
)
AS
BEGIN
	SELECT 	[dbo].[MtrReport].[MtrReportID],
			[SyncboxID],
			[StartTime],
			[DataCenter],
			[HopNumber],
			[HostName],
			[PacketLoss],
			[PacketsSent],
			[LastPingMS],
			[AvgPingMS],
			[BestPingMS],
			[WorstPingMS],
			[StandardDev]
	FROM	[dbo].[MtrReport]
			INNER JOIN [dbo].[MtrHop]
		ON	[dbo].[MtrReport].[MtrReportID] = [dbo].[MtrHop].[MtrReportID]
	WHERE 	[SyncboxID] = @SyncboxID
	AND		[MtrReport].[StartTime] >= @StartDatetime
	AND		[MtrReport].[StartTime] <= @EndDatetime
	ORDER BY [MtrReportID]
END
GO
