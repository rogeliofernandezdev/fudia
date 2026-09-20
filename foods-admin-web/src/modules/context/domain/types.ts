export type OrgSummary = {
  id: string;
  name: string;
  active: boolean;
  locationCount: number;
};

export type LocationSummary = {
  id: string;
  name: string;
  code: string;
  active: boolean;
};

export type SwitchContextInput = {
  organizationId: string;
  locationId: string;
};

export type ContextResponse = {
  organization: { id: string; name: string };
  location: { id: string; name: string };
};
