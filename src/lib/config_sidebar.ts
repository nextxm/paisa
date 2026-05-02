export const MOBILE_CONFIG_SIDEBAR_MAX_WIDTH = 1023;

export function shouldCollapseConfigSidebarOnNavigate(
  viewportWidth: number | null | undefined
): boolean {
  return typeof viewportWidth === "number" && viewportWidth <= MOBILE_CONFIG_SIDEBAR_MAX_WIDTH;
}
