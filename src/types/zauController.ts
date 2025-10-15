export interface IZauRole {
	name: string;
	code: string;
	order: number;
	class: string;
}

interface IZauCertification {
	name: string;
	code: string;
	order: number;
	class: string;
	facility: string;
}

interface IZauAbsence {
	controller: number;
	expirationDate: Date;
	deleted: boolean;
}

export interface IZauController {
	cid: number;
	fname: string;
	lname: string;
	email: string;
	rating: number;
	oi?: string | null;
	broadcast: boolean;
	member: boolean;
	vis: boolean;
	homeFacility?: string;
	bio: string;
	avatar?: string;
	joinDate?: Date | null;
	removalDate?: Date | null;
	prefName: boolean;
	discordInfo?: {
		clientId: string;
		accessToken: string;
		refreshToken: string;
		tokenType: string;
		expires: Date;
	};
	discord?: string;
	idsToken?: string;
	certCodes: string[];
	roleCodes: string[];
	trainingMilestones: [];

	// Virtual Properties
	isMem: boolean;
	isManagement: boolean;
	isSeniorStaff: boolean;
	isStaff: boolean;
	isIns: boolean;
	ratingShort: string;
	ratingLong: string;
	certCodeList: string[];

	roles: IZauRole[];
	certifications: IZauCertification[];
	absence: IZauAbsence[];
}
