export namespace db {
	
	export class AccountRecord {
	    id: string;
	    provider: string;
	    displayName: string;
	    email: string;
	    accessToken: string;
	    refreshToken: string;
	    tokenExpiry: string;
	    usedSpace: number;
	    totalSpace: number;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AccountRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.provider = source["provider"];
	        this.displayName = source["displayName"];
	        this.email = source["email"];
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.tokenExpiry = source["tokenExpiry"];
	        this.usedSpace = source["usedSpace"];
	        this.totalSpace = source["totalSpace"];
	        this.active = source["active"];
	    }
	}
	export class ActivityRecord {
	    id: number;
	    fileId: string;
	    fileName: string;
	    action: string;
	    details: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new ActivityRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fileId = source["fileId"];
	        this.fileName = source["fileName"];
	        this.action = source["action"];
	        this.details = source["details"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class FileRecord {
	    id: string;
	    name: string;
	    size: number;
	    isFolder: boolean;
	    parentId: string;
	    provider: string;
	    accountId: string;
	    physicalId: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    modifiedAt: any;
	    starred: boolean;
	    shared: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.isFolder = source["isFolder"];
	        this.parentId = source["parentId"];
	        this.provider = source["provider"];
	        this.accountId = source["accountId"];
	        this.physicalId = source["physicalId"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.modifiedAt = this.convertValues(source["modifiedAt"], null);
	        this.starred = source["starred"];
	        this.shared = source["shared"];
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
	export class SyncTask {
	    id: string;
	    localPath: string;
	    targetFolderId: string;
	    accountId: string;
	    syncMode: string;
	    enabled: boolean;
	    lastSync: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.localPath = source["localPath"];
	        this.targetFolderId = source["targetFolderId"];
	        this.accountId = source["accountId"];
	        this.syncMode = source["syncMode"];
	        this.enabled = source["enabled"];
	        this.lastSync = source["lastSync"];
	    }
	}

}

export namespace provider {
	
	export class SharePermission {
	    id: string;
	    type: string;
	    role: string;
	    emailAddress?: string;
	    displayName?: string;
	
	    static createFrom(source: any = {}) {
	        return new SharePermission(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.role = source["role"];
	        this.emailAddress = source["emailAddress"];
	        this.displayName = source["displayName"];
	    }
	}

}

