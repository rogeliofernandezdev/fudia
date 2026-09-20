export type Module = {
  key: string;
  name: string;
  description: string;
  icon: string;
  category: string;
  active: boolean;
};

export type ModulesResponse = { modules: Module[] };

export type ToggleModuleInput = { key: string; active: boolean };
