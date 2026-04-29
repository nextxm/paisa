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

  const selectedLink =
    _.find(links, (link) => normalizedPath === link.href) ||
    _.find(links, (link) => !_.isEmpty(link.children) && normalizedPath.startsWith(link.href)) ||
    null;

  if (!selectedLink || _.isEmpty(selectedLink.children)) {
    return {
      selectedLink,
      selectedSubLink: null,
      selectedSubSubLink: null
    };
  }

  const selectedSubLink =
    _.find(
      selectedLink.children,
      (subLink) => normalizedPath === selectedLink.href + subLink.href
    ) ||
    _.find(selectedLink.children, (subLink) =>
      normalizedPath.startsWith(selectedLink.href + subLink.href)
    ) ||
    null;

  if (!selectedSubLink || _.isEmpty(selectedSubLink.children)) {
    return {
      selectedLink,
      selectedSubLink,
      selectedSubSubLink: null
    };
  }

  const selectedSubSubLink =
    _.find(selectedSubLink.children, (subSubLink) =>
      normalizedPath.startsWith(selectedLink.href + selectedSubLink.href + subSubLink.href)
    ) || null;

  return {
    selectedLink,
    selectedSubLink,
    selectedSubSubLink
  };
}

export function resolveNavbarSelectionTyped<T extends NavLink>(
  links: T[],
  normalizedPath: string | null | undefined
): NavbarSelectionFor<T> {
  return resolveNavbarSelection(links, normalizedPath) as NavbarSelectionFor<T>;
}
