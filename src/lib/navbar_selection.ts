import _ from "lodash";

export interface NavLink {
  href: string;
  children?: NavLink[];
}

export interface NavbarSelection {
  selectedLink: NavLink | null;
  selectedSubLink: NavLink | null;
  selectedSubSubLink: NavLink | null;
}

export interface NavbarSelectionFor<T extends NavLink> {
  selectedLink: T | null;
  selectedSubLink: T | null;
  selectedSubSubLink: T | null;
}

export function resolveHref(parentHref: string, childHref: string): string {
  if (childHref.startsWith("/../")) {
    return childHref.replace("/..", "");
  }
  return parentHref + childHref;
}

export function resolveNavbarSelection(
  links: NavLink[],
  normalizedPath: string | null | undefined
): NavbarSelection {
  if (!normalizedPath) {
    return {
      selectedLink: null,
      selectedSubLink: null,
      selectedSubSubLink: null
    };
  }

  // Exact leaf matches first across all hierarchies
  for (const link of links) {
    if (link.href === normalizedPath && _.isEmpty(link.children)) {
      return { selectedLink: link, selectedSubLink: null, selectedSubSubLink: null };
    }

    if (!_.isEmpty(link.children)) {
      for (const subLink of link.children!) {
        const subHref = resolveHref(link.href, subLink.href);

        if (!_.isEmpty(subLink.children)) {
          // Check 3rd level exact match
          for (const subSubLink of subLink.children!) {
            const subSubHref = resolveHref(subHref, subSubLink.href);
            if (normalizedPath === subSubHref) {
              return {
                selectedLink: link,
                selectedSubLink: subLink,
                selectedSubSubLink: subSubLink
              };
            }
          }
        } else {
          // Check 2nd level exact match
          if (normalizedPath === subHref) {
            return {
              selectedLink: link,
              selectedSubLink: subLink,
              selectedSubSubLink: null
            };
          }
        }
      }
    }
  }

  // Prefix matches (deepest first)
  for (const link of links) {
    if (!_.isEmpty(link.children)) {
      for (const subLink of link.children!) {
        const subHref = resolveHref(link.href, subLink.href);

        if (!_.isEmpty(subLink.children)) {
          for (const subSubLink of subLink.children!) {
            const subSubHref = resolveHref(subHref, subSubLink.href);
            if (subSubHref !== "/" && normalizedPath.startsWith(subSubHref)) {
              return {
                selectedLink: link,
                selectedSubLink: subLink,
                selectedSubSubLink: subSubLink
              };
            }
          }
        } else {
          if (subHref !== "/" && normalizedPath.startsWith(subHref)) {
            return {
              selectedLink: link,
              selectedSubLink: subLink,
              selectedSubSubLink: null
            };
          }
        }
      }
    }
  }

  // Top level fallback
  const selectedLink =
    _.find(links, (link) => normalizedPath === link.href) ||
    _.find(
      links,
      (link) =>
        !_.isEmpty(link.children) && link.href !== "/" && normalizedPath.startsWith(link.href)
    ) ||
    null;

  return {
    selectedLink,
    selectedSubLink: null,
    selectedSubSubLink: null
  };
}

export function resolveNavbarSelectionTyped<T extends NavLink>(
  links: T[],
  normalizedPath: string | null | undefined
): NavbarSelectionFor<T> {
  return resolveNavbarSelection(links, normalizedPath) as NavbarSelectionFor<T>;
}
