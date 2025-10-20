import type {
	IRoleResponse,
	IVatusaControllerResponse,
	IZauControllerResponse,
} from './types/apiResponses.js';

const zauApi = (uri: string, options: RequestInit = {}) => {
	if (!uri.startsWith('/')) uri = '/' + uri;

	return fetch(`${process.env['ZAU_API_URL']}/controller${uri}`, {
		...options,
		headers: {
			Authorization: `Bearer ${process.env['ZAU_API_KEY']}`,
			'Content-Type': 'application/json',
		},
	});
};

export async function vatusaSync() {
	try {
		console.log(`\n\n⏳ Starting sync . . .`);
		const start = performance.now();

		const sixMonthsAgo = new Date();
		sixMonthsAgo.setDate(new Date().getDate() - 182);

		const usaMark = performance.now();
		const { data: vatusaData }: IVatusaControllerResponse = (await fetch(
			`https://api.vatusa.net/v2/facility/ZAU/roster/both?apikey=${process.env['VATUSA_API_KEY']}`,
			{
				method: 'GET',
			},
		).then((response) => response.json())) as IVatusaControllerResponse;

		console.log(`VATUSA API took ${(performance.now() - usaMark).toFixed(2)}ms`);

		const zauCMark = performance.now();
		const { data: zauData }: IZauControllerResponse = (await zauApi('/', {
			method: 'GET',
		}).then((response) => response.json())) as IZauControllerResponse;
		console.log(`ZAU Controller API took ${(performance.now() - zauCMark).toFixed(2)}ms`);

		const zauRMark = performance.now();
		const { data: zauRoles }: IRoleResponse = (await zauApi(`/role`, { method: 'GET' }).then(
			(response) => response.json(),
		)) as IRoleResponse;
		console.log(`ZAU Role API took ${(performance.now() - zauRMark).toFixed(2)}ms`);

		console.log(`API calls took ${(performance.now() - start).toFixed(2)}ms to complete.`);

		const availableRoles = zauRoles.map((role) => role.code);

		const allZauControllers = [...zauData.home, ...zauData.visiting, ...zauData.removed];

		const zauAllCids = allZauControllers.map((c) => c.cid); // everyone in database

		const zauMemberCids = allZauControllers.filter((c) => c.member).map((c) => c.cid); // only active (home AND vis) controllers (member: true)
		const zauNonMemberCids = allZauControllers.filter((c) => !c.member).map((c) => c.cid); // only inactive controllers (member: false)

		const zauHomeCids = zauData.home.map((c) => c.cid); // only home controllers (vis: false)
		const zauVisitCids = zauData.visiting.map((c) => c.cid); // only visiting controllers (vis: true)

		const zauCertRemovalCids = zauData.removed
			.filter(
				(c) =>
					c.removalDate &&
					new Date(c.removalDate) < sixMonthsAgo &&
					c.certCodes &&
					c.certCodes.length > 0,
			)
			.map((c) => c.cid); // certs to remove from ex-members

		const vatusaAllCids = vatusaData.map((c) => c.cid); // all vatusa controllers returned
		const vatusaHomeCids = vatusaData.filter((c) => c.membership === 'home').map((c) => c.cid); // only home controllers
		const vatusaVisitCids = vatusaData.filter((c) => c.membership !== 'home').map((c) => c.cid); // all non-home controllers

		const toBeAdded = vatusaAllCids.filter((cid) => !zauAllCids.includes(cid));
		const makeNonMember = zauMemberCids.filter((cid) => !vatusaAllCids.includes(cid));
		const makeMember = zauNonMemberCids.filter((cid) => vatusaAllCids.includes(cid));
		const makeVisitor = zauHomeCids.filter((cid) => vatusaVisitCids.includes(cid));
		const makeHome = zauVisitCids.filter((cid) => vatusaHomeCids.includes(cid));

		console.log(`Members to be added: ${toBeAdded.join(', ')}`);
		console.log(`Members to be removed: ${makeNonMember.join(', ')}`);
		console.log(`Controllers to be made member: ${makeMember.join(', ')}`);
		console.log(`Controllers to be made visitor: ${makeVisitor.join(', ')}`);
		console.log(`Controllers to be made home controller: ${makeHome.join(', ')}`);
		console.log(`Certs removed after 6 months of being removed: ${zauCertRemovalCids.join(', ')}`);

		// Add users to ZAU database from VATUSA data if we don't have the local use
		for (const cid of toBeAdded) {
			const user = vatusaData.find((c) => c.cid === cid);
			if (!user) continue;

			// Convert VATUSA role into ZAU roles
			const assignableRoles = user.roles
				.filter((roles) => availableRoles.includes(roles.role.toLowerCase()))
				.map((role) => role.role.toLowerCase());

			const isVisitor = user.membership !== 'home';

			const userData = {
				cid,
				fname: user.fname,
				lname: user.lname,
				rating: user.rating,
				home: user.facility,
				email: user.email,
				broadcast: user.flag_broadcastOptedIn,
				member: true,
				vis: isVisitor,
				roleCodes: !isVisitor ? assignableRoles : [],
				createdAt: new Date(),
				joinDate: user.facility_join,
			};

			await zauApi(`/${cid}`, {
				method: 'POST',
				body: JSON.stringify(userData),
			}).catch((err) => {
				console.log('Error creating new user for', cid, ':', err);
			});
		}

		// Update ratings in ZAU database from VATUSA data
		for (const sUser of vatusaData) {
			const zUser = allZauControllers.find((c) => c.cid === sUser.cid);

			if (zUser && zUser.rating !== sUser.rating) {
				await zauApi(`/${zUser.cid}/rating`, {
					method: 'PUT',
					body: JSON.stringify({ rating: sUser.rating }),
				}).catch((err) => {
					console.log('Error updating rating for', sUser.cid, ':', err);
				});
			}
		}

		// Convert non-member users into members from VATUSA data
		for (const cid of makeMember) {
			await zauApi(`/${cid}/member`, {
				method: 'PUT',
				body: JSON.stringify({ member: true }),
			}).catch((err) => {
				console.log('Error making', cid, 'a member:', err);
			});
		}

		// Convert member users into non-members from VATUSA data
		for (const cid of makeNonMember) {
			await zauApi(`/${cid}/member`, {
				method: 'PUT',
				body: JSON.stringify({ member: false }),
			}).catch((err) => {
				console.log('Error removing', cid, 'as a member:', err);
			});
		}

		// Convert to visitor from VATUSA data
		for (const cid of makeVisitor) {
			await zauApi(`/${cid}/visit`, { method: 'PUT', body: JSON.stringify({ vis: true }) }).catch(
				(err) => {
					console.log('Error making', cid, 'a visitor:', err);
				},
			);
		}

		// Convert to home controller from VATUSA data
		for (const cid of makeHome) {
			await zauApi(`/${cid}/visit`, { method: 'PUT', body: JSON.stringify({ vis: false }) }).catch(
				(err) => {
					console.log('Error making', cid, 'a home controller:', err);
				},
			);
		}

		for (const cid of zauCertRemovalCids) {
			await zauApi(`/remove-cert/${cid}`, { method: 'PUT', body: JSON.stringify({}) }).catch(
				(err) => {
					console.log('Error removing cert for', cid, ':', err);
				},
			);
		}

		console.log(`\n✅ Done!  Finished in ${(performance.now() - start).toFixed(2)}ms`);
	} catch (err) {
		console.error(err);
	}
}
