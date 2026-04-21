// Re-export from the main cloudflare module for backwards compatibility
// sst.cloudflare.x.StaticSite still works but is deprecated
export { StaticSiteV2 as StaticSite } from "../static-site-v2.js";
export type { StaticSiteV2Args as StaticSiteArgs } from "../static-site-v2.js";
