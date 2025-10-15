interface IVatusaRole {
	id: number;
	cid: number;
	facility: string;
	role: string;
	created_at: Date;
}

export interface IVatusaController {
	cid: number;
	fname: string;
	lname: string;
	email: string;
	facility: string;
	rating: number;
	created_at: Date;
	updated_at: Date;
	flag_needbasic: boolean;
	flag_xferOverride: boolean;
	facility_join: Date;
	flag_homecontroller: true;
	lastactivity: Date;
	flag_broadcastOptedIn: boolean;
	flag_preventStaffAssign: any;
	discord_id: number;
	last_cert_sync: Date;
	flag_nameprivacy: boolean;
	last_compentency_date: any;
	promotion_eligible: false;
	transfer_eligible: any;
	roles: IVatusaRole[];
	rating_short: string;
	isMentor: boolean;
	isSupIns: false;
	last_promotion: Date;
	membership: 'home' | 'visit';
}
