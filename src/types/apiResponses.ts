import type { IVatusaController } from './vatusaController.js';
import type { IZauController, IZauRole } from './zauController.js';

export interface IRoleResponse {
	ret_det: {
		code: number;
		message: string;
	};
	data: IZauRole[];
}

export interface IZauControllerResponse {
	ret_det: {
		code: number;
		message: string;
	};
	data: {
		home: IZauController[];
		visiting: IZauController[];
		removed: IZauController[];
	};
}

export interface IVatusaControllerResponse {
	testing: boolean;
	data: IVatusaController[];
}
