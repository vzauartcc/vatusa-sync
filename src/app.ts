import { Cron } from 'croner';
import { vatusaSync } from './sync.js';

if (!process.env['ZAU_API_URL'] || !process.env['ZAU_API_KEY'] || !process.env['VATUSA_API_KEY']) {
	console.error(
		'Missing at least one environment variable. Check to make sure the follow are set: "VATUSA_API_KEY", "ZAU_API_URL", "ZAU_API_KEY',
	);
	process.exit(4);
}
const task = new Cron('*/10 * * * *', () => vatusaSync());

console.log(`Performing startup run. Next scheduled run is at: ${task.nextRun()}`);
vatusaSync();
