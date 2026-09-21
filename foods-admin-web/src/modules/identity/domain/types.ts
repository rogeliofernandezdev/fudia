export type Assignment={roleId:string;roleName?:string;locationId:string;locationName?:string};
export type User={id:string;fullName:string;email:string;active:boolean;platformAdmin:boolean;assignments:Assignment[]};
export type Role={
  id:string;
  name:string;
  systemKey:string|null;
  description:string;
  menuAccess:string[];
  permissions:string[];
  active:boolean;
  userCount:number;
};
export type Location={id:string;name:string;active:boolean};
export type PermissionGroup={group:string;items:{value:string;label:string}[]};
export type UserDraft={id?:string;fullName:string;email:string;password:string;assignments:Assignment[]};
export type RoleDraft={id?:string;systemKey?:string|null;name:string;description:string;menuAccess?:string[];permissions:string[]};

export type MyProfile={fullName:string;email:string};
export type MyProfileDraft={fullName:string;currentPassword:string;newPassword:string};
