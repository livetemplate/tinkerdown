export namespace main {
	
	export class RouteInfo {
	    pattern: string;
	    filePath: string;
	
	    static createFrom(source: any = {}) {
	        return new RouteInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.filePath = source["filePath"];
	    }
	}

}

