export interface DashboardLinks {
  menuLinks: MenuLink[];
  externalLinks?: any[];
  quickLinks?: Link[];
  documentationItems?: Link[];
  contentPlugins?: ContentPlugin[];
}

export interface ContentPlugin {
  id: string;
  route: string;
  element: string;
  bundle: string;
  enabled?: boolean;
  contractVersion?: number;
  namespaced?: boolean;
  replaces?: string;
}

export interface MenuLink {
  type: string;
  link: string;
  text: string;
  icon: string;
}

export interface Link {
  text: string;
  desc: string;
  link: string;
}
