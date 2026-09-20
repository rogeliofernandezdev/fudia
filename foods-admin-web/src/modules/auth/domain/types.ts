export type AuthenticatedUser = {
  id: string;
  name: string;
  platformAdmin: boolean;
};

export type LoginCredentials = {
  email: string;
  password: string;
};
