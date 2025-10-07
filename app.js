import axios from 'axios';
import dotenv from 'dotenv';
import schedule from 'node-schedule';
import { performance } from 'perf_hooks';

dotenv.config();

const syncRoster = async () => {

	const zabApi = axios.create({
		baseURL: process.env.ZAB_API_URL,
		headers: {
			'Authorization': `Bearer ${process.env.ZAB_API_KEY}`
		}
	})

	const start = performance.now();

	const sixMonthsAgo = new Date();
	sixMonthsAgo.setDate(sixMonthsAgo.getDate() - 182);

	console.log(`Syncing Roster...`);

	const { data: vatusaData } = await axios.get(`https://api.vatusa.net/v2/facility/ZAU/roster/both?apikey=${process.env.VATUSA_API_KEY}`).catch(console.error);
	const { data: zauData } = await axios.get(`${process.env.ZAB_API_URL}/controller`);
	const allZauControllers = [...zauData.data.home, ...zauData.data.visiting];
	const nonZauControllers = zauData.data.removed;
	const { data: zauRoles } = await axios.get(`${process.env.ZAB_API_URL}/controller/role`);
	const availableRoles = zauRoles.data.map(role => role.code);

	const zauControllers = allZauControllers.map(c => c.cid); // everyone in db
	const zauMembers = allZauControllers.filter(c => c.member).map(c => c.cid); // only member: true
	const zauNonMembers = allZauControllers.filter(c => !c.member).map(c => c.cid); // only member: false
	const zauHomeControllers = zauData.data.home.map(c => c.cid); // only vis: false
	const zauVisitingControllers = zauData.data.visiting.map(c => c.cid); // only: vis: true
	const zauCertRemoval = nonZauControllers.filter(c => c.removalDate && new Date(c.removalDate) < sixMonthsAgo && (c.certCodes && c.certCodes.length > 0)).map(c => c.cid);

	const vatusaControllers = vatusaData.data.map(c => c.cid); // all controllers returned by VATUSA
	const vatusaHomeControllers = vatusaData.data.filter(c => c.membership === 'home').map(c => c.cid); // only membership: home
	const vatusaVisitingControllers = vatusaData.data.filter(c => c.membership !== 'home').map(c => c.cid); // only membership: !home

	const toBeAdded = vatusaControllers.filter(cid => !zauControllers.includes(cid));
	const makeNonMember = zauMembers.filter(cid => !vatusaControllers.includes(cid));
	const makeMember = zauNonMembers.filter(cid => vatusaControllers.includes(cid));
	const makeVisitor = zauHomeControllers.filter(cid => vatusaVisitingControllers.includes(cid));
	const makeHome = zauVisitingControllers.filter(cid => vatusaHomeControllers.includes(cid));

	console.log(`Members to be added: ${toBeAdded.join(', ')}`);
	console.log(`Members to be removed: ${makeNonMember.join(', ')}`);
	console.log(`Controllers to be made member: ${makeMember.join(', ')}`);
	console.log(`Controllers to be made visitor: ${makeVisitor.join(', ')}`);
	console.log(`Controllers to be made home controller: ${makeHome.join(', ')}`);
	console.log(`Certs removed after 6 months of being removed: ${zauCertRemoval.join(', ')}`);

	const vatusaObject = {};

	for(const user of vatusaData.data) {
		vatusaObject[user.cid] = user;
	}

	for (const cid of toBeAdded) {
		const user = vatusaObject[cid];

		const assignableRoles = user.roles.filter(role => availableRoles.includes(role.role.toLowerCase())).map(role => role.role.toLowerCase());

		const isVisitor = (user.membership === 'home') ? false : true

		const userData = {
			fname: user.fname,
			lname: user.lname,
			cid: user.cid,
			rating: user.rating,
			home: user.facility,
			email: user.email,
			broadcast: user.flag_broadcastOptedIn,
			member: true,
			vis: isVisitor,
			roleCodes: !isVisitor ? assignableRoles : [],
			createdAt: new Date(),
			joinDate: user.facility_join
		}

		await zabApi.post(`/controller/${user.cid}`, userData);
	}
	for (const cid of vatusaControllers){
		const user = vatusaObject[cid];
		await zabApi.put(`/controller/${cid}/rating`, {rating: user.rating});
	}

	for (const cid of makeMember) {
		await zabApi.put(`/controller/${cid}/member`, {member: true});
	}

	for (const cid of makeNonMember) {
		await zabApi.put(`/controller/${cid}/member`, {member: false});
	}

	for (const cid of makeVisitor) {
		await zabApi.put(`/controller/${cid}/visit`, {vis: true});
	}

	for (const cid of makeHome) {
		await zabApi.put(`/controller/${cid}/visit`, {vis: false});
	}

	for (const cid of zauCertRemoval) {
		await zabApi.put(`/controller/remove-cert/${cid}`, { certCodes: {} })
	  }

	console.log(`...Done!\nFinished in ${Math.round(performance.now() - start)/1000}s\n---`);
}


(() => {
	syncRoster();
	schedule.scheduleJob('*/10 * * * *', syncRoster);
})();
