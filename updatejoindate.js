import axios from 'axios';
import dotenv from 'dotenv';
import schedule from 'node-schedule';
import { performance } from 'perf_hooks';
import { MongoClient } from "mongodb";

dotenv.config();

const syncRoster = async () => {

	const zabApi = axios.create({
		baseURL: process.env.ZAB_API_URL,
		headers: {
			'Authorization': `Bearer ${process.env.ZAB_API_KEY}`
		}
	})

	const start = performance.now();

	console.log(`Syncing Roster...`);

	const { data: vatusaData } = await axios.get(`https://api.vatusa.net/v2/facility/ZAU/roster/both?apikey=${process.env.VATUSA_API_KEY}`).catch(console.error);
	const { data: zabData } = await axios.get(`${process.env.ZAB_API_URL}/controller`);
	const allZabControllers = [...zabData.data.home, ...zabData.data.visiting];
	const { data: zabRoles } = await axios.get(`${process.env.ZAB_API_URL}/controller/role`);
	const availableRoles = zabRoles.data.map(role => role.code);

	const zabControllers = allZabControllers.map(c => c.cid); // everyone in db
	const zabMembers = allZabControllers.filter(c => c.member).map(c => c.cid); // only member: true
	const zabNonMembers = allZabControllers.filter(c => !c.member).map(c => c.cid); // only member: false
	const zabHomeControllers = zabData.data.home.map(c => c.cid); // only vis: false
	const zabVisitingControllers = zabData.data.visiting.map(c => c.cid); // only: vis: true

	const vatusaControllers = vatusaData.data.map(c => c.cid); // all controllers returned by VATUSA
	const vatusaHomeControllers = vatusaData.data.filter(c => c.membership === 'home').map(c => c.cid); // only membership: home
	const vatusaVisitingControllers = vatusaData.data.filter(c => c.membership !== 'home').map(c => c.cid); // only membership: !home

	const toBeAdded = vatusaControllers; //add to every user
	const makeNonMember = zabMembers.filter(cid => !vatusaControllers.includes(cid));
	const makeMember = zabNonMembers.filter(cid => vatusaControllers.includes(cid));
	const makeVisitor = zabHomeControllers.filter(cid => vatusaVisitingControllers.includes(cid));
	const makeHome = zabVisitingControllers.filter(cid => vatusaHomeControllers.includes(cid));

	console.log(`Members to be added: ${toBeAdded.join(', ')}`);
	console.log(`Members to be removed: ${makeNonMember.join(', ')}`);
	console.log(`Controllers to be made member: ${makeMember.join(', ')}`);
	console.log(`Controllers to be made visitor: ${makeVisitor.join(', ')}`);
	console.log(`Controllers to be made home controller: ${makeHome.join(', ')}`);

	const vatusaObject = {};

	for(const user of vatusaData.data) {
		vatusaObject[user.cid] = user;
	}

	for (const cid of toBeAdded) {
		const user = vatusaObject[cid];


		const userData = {
			createdAt: user.facility_join
		}


// Replace the uri string with your MongoDB deployment's connection string.

		const uri = process.env.MONGO_URI;

		const client = new MongoClient(uri);
		const database = client.db("test"); //db name! NOT THE CLUSTER NAME!
		const users = database.collection("users");
		const filter = { cid: cid };
		const updateDate = {
			$set: {
				createdAt: user.facility_join
			},
		}
		await users.updateOne(filter, updateDate)

	}



	console.log(`...Done!\nFinished in ${Math.round(performance.now() - start)/1000}s\n---`);
}

(() => {
	syncRoster();
	schedule.scheduleJob('*/10 * * * *', syncRoster);
})();