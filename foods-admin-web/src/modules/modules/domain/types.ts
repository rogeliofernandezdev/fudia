export type ModuleAvailability = "ready" | "development" | "planned";

export type Module = {
  key: string;
  name: string;
  description: string;
  icon: string;
  category: string;
  availability: ModuleAvailability;
  active: boolean;
};

export type ModulesResponse = { modules: Module[] };

export type ToggleModuleInput = { key: string; active: boolean };
