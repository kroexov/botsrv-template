-- =============================================================================
-- Diagram Name: apisrv
-- Created on: 7/29/2025 11:06:02 PM
-- Diagram Version: 
-- =============================================================================

CREATE TABLE "users" (
	"userId" SERIAL NOT NULL,
	"nickname" varchar(128) NOT NULL,
	"createdAt" timestamp with time zone NOT NULL DEFAULT now(),
	"statusId" int4 NOT NULL,
	CONSTRAINT "users_pkey" PRIMARY KEY("userId")
);

CREATE TABLE "places" (
	"placeId" int4 NOT NULL,
	"placeName" text NOT NULL,
	"placePriority" int4 NOT NULL,
	"userId" int4,
	PRIMARY KEY("placeId")
);


ALTER TABLE "places" ADD CONSTRAINT "Ref_places_to_users" FOREIGN KEY ("userId")
	REFERENCES "users"("userId")
	MATCH SIMPLE
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;


