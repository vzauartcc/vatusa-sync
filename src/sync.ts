import type { IVatusaControllerResponse, IZauControllerResponse } from './types/apiResponses.js';
import type { IZauRole } from './types/zauController.js';

const zauApi = (uri: string, options: RequestInit = {}) => {
	if (!uri.startsWith('/')) uri = '/' + uri;

	return fetch(`${process.env['ZAU_API_URL']}${uri}`, {
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

		const today = new Date();
		const sixMonthsAgo = new Date(
			today.getUTCFullYear(),
			today.getUTCMonth(),
			today.getUTCDate() - 182,
		);

		const usaMark = performance.now();
		const vatusaResponse = await fetch(
			`https://api.vatusa.net/v2/facility/ZAU/roster/both?apikey=${process.env['VATUSA_API_KEY']}`,
			{
				method: 'GET',
			},
		);
		if (!vatusaResponse.ok) {
			console.error(
				'Failed to fetch VATUSA data',
				vatusaResponse.status,
				vatusaResponse.statusText,
			);
			return;
		}

		const { data: vatusaData }: IVatusaControllerResponse =
			(await vatusaResponse.json()) as IVatusaControllerResponse;

		console.debug(`VATUSA API took ${(performance.now() - usaMark).toFixed(2)}ms`);

		const zauCMark = performance.now();
		const zauResponse = await zauApi('/user', { method: 'GET' });
		if (!zauResponse.ok) {
			console.error('Failed to fetch ZAU data', zauResponse.status, zauResponse.statusText);
		}
		const zauData = (await zauResponse.json()) as IZauControllerResponse;
		console.debug(`ZAU Controller API took ${(performance.now() - zauCMark).toFixed(2)}ms`);

		const zauRMark = performance.now();
		const zauRoles: IZauRole[] = (await (
			await zauApi(`/controller/role`, {
				method: 'GET',
			})
		).json()) as IZauRole[];
		console.debug(`ZAU Role API took ${(performance.now() - zauRMark).toFixed(2)}ms`);

		console.debug(`API calls took ${(performance.now() - start).toFixed(2)}ms to complete.`);

		const availableRoles = zauRoles.map((role) => role.code);

		const allZauControllers = [...zauData.home, ...zauData.visiting, ...zauData.removed];

		const zauMemberCids = allZauControllers.filter((c) => c.member).map((c) => c.cid); // only active (home AND vis) controllers (member: true)

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

		const makeNonMember = zauMemberCids.filter((cid) => !vatusaAllCids.includes(cid));

		let added = 0;
		let madeMember = 0;
		let madeVisitor = 0;
		let updatedCore = 0;
		let updatedRating = 0;
		let removedMember = 0;
		let certsRemoved = 0;

		for (const vUser of vatusaData) {
			const zUser = allZauControllers.find((z) => z.cid === vUser.cid);

			const isVisitor = vUser.membership !== 'home';

			// Add user to database
			if (!zUser) {
				const assignableRoles = vUser.roles
					.filter((roles) => availableRoles.includes(roles.role.toLowerCase()))
					.map((role) => role.role.toLowerCase());

				await zauApi(`/controller/${vUser.cid}`, {
					method: 'POST',
					body: JSON.stringify({
						cid: vUser.cid,
						fname: vUser.fname,
						lname: vUser.lname,
						rating: vUser.rating,
						home: vUser.facility,
						email: vUser.email,
						broadcast: vUser.flag_broadcastOptedIn,
						member: true,
						vis: isVisitor,
						roleCodes: !isVisitor ? assignableRoles : [],
						createdAt: new Date(),
						joinDate: vUser.facility_join,
						prefName: vUser.flag_nameprivacy,
					}),
				})
					.then(() => {
						added++;
					})
					.catch((err) => {
						console.log(`Error creating new user for ${vUser.cid}: ${err}`);
					});
			} else {
				// Update user if core info changed
				if (
					vUser.fname !== zUser.fname ||
					vUser.lname !== zUser.lname ||
					vUser.email !== zUser.email ||
					vUser.flag_broadcastOptedIn !== zUser.broadcast ||
					vUser.flag_nameprivacy !== zUser.prefName
				) {
					await zauApi(`/user/${zUser.cid}`, {
						method: 'PATCH',
						body: JSON.stringify({
							fname: vUser.fname,
							lname: vUser.lname,
							email: vUser.email,
							broadcast: vUser.flag_broadcastOptedIn,
							prefName: vUser.flag_nameprivacy,
						}),
					})
						.then(() => {
							updatedCore++;
						})
						.catch((err) => console.log(`Error update ${zUser.cid}'s core details: ${err}`));
				}

				// Update membership if necessary
				if (!zUser.member) {
					await zauApi(`/controller/${zUser.cid}/member`, {
						method: 'PATCH',
						body: JSON.stringify({ member: true, joinDate: vUser.facility_join }),
					})
						.then(() => {
							madeMember++;
						})
						.catch((err) => {
							console.log(`Error making ${zUser.cid} a member: ${err}`);
						});
				}

				// Update visiting status if necessary
				if (zUser.vis !== isVisitor) {
					await zauApi(`/controller/${zUser.cid}/visit`, {
						method: 'PATCH',
						body: JSON.stringify({ vis: isVisitor }),
					})
						.then(() => {
							madeVisitor++;
						})
						.catch((err) => {
							console.log(`Error making ${zUser.cid} a visitor: ${err}`);
						});
				}

				// Update rating if necessary
				if (zUser.rating !== vUser.rating) {
					await zauApi(`/controller/${zUser.cid}/rating`, {
						method: 'PATCH',
						body: JSON.stringify({ rating: vUser.rating }),
					})
						.then(() => {
							updatedRating++;
						})
						.catch((err) => {
							console.log(`Error updating rating for ${vUser.cid}: ${err}`);
						});
				}
			}
		}

		// Convert member users into non-member users from VATUSA data
		for (const cid of makeNonMember) {
			await zauApi(`/controller/${cid}/member`, {
				method: 'PATCH',
				body: JSON.stringify({ member: false }),
			})
				.then(() => {
					removedMember++;
				})
				.catch((err) => {
					console.log('Error removing', cid, 'as a member:', err);
				});
		}

		for (const cid of zauCertRemovalCids) {
			await zauApi(`/controller/${cid}/remove-cert`, {
				method: 'PATCH',
				body: JSON.stringify({}),
			})
				.then(() => {
					certsRemoved++;
				})
				.catch((err) => {
					console.log('Error removing cert for', cid, ':', err);
				});
		}

		console.log(`Added ${added} new users.`);
		console.log(`Made ${madeMember} users into members.`);
		console.log(`Changed ${madeVisitor} users visiting status.`);
		console.log(`Updated ${updatedCore} users' core data.`);
		console.log(`Updated ${updatedRating} user ratings.`);
		console.log(`Removed ${removedMember} controllers' membership status.`);
		console.log(`Removed ${certsRemoved} expired certs for non-member controllers.`);

		console.log(`\n✅ Done!  Finished in ${(performance.now() - start).toFixed(2)}ms`);
	} catch (err) {
		console.error(err);
	}
}
