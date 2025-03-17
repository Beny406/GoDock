export namespace main {
	
	export class WmCtrlInstance {
	    windowId: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new WmCtrlInstance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.windowId = source["windowId"];
	        this.name = source["name"];
	    }
	}
	export class DesktopFile {
	    name: string;
	    iconPath: string;
	    instances: WmCtrlInstance[];
	    execPath: string;
	    wmClass: string;
	
	    static createFrom(source: any = {}) {
	        return new DesktopFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.iconPath = source["iconPath"];
	        this.instances = this.convertValues(source["instances"], WmCtrlInstance);
	        this.execPath = source["execPath"];
	        this.wmClass = source["wmClass"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

