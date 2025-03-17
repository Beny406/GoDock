export namespace main {
	
	export class AppInfo {
	    name: string;
	    iconPath: string;
	    runningId: string;
	    execPath: string;
	    wmClass: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.iconPath = source["iconPath"];
	        this.runningId = source["runningId"];
	        this.execPath = source["execPath"];
	        this.wmClass = source["wmClass"];
	    }
	}

}

