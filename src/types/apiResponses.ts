import type { IVatusaController } from './vatusaController.js';
import type { IZauController } from './zauController.js';

export interface IZauControllerResponse {
	home: IZauController[];
	visiting: IZauController[];
	removed: IZauController[];
}

export interface IVatusaControllerResponse {
	testing: boolean;
	data: IVatusaController[];
}
