// Remembers the last department/company picked on ANY of the cascading
// org-scope pickers (Applications, Attendance review, Journal review,
// Scores, Monitoring visits, Admin Vacancies) so navigating between them
// doesn't reset both dropdowns every time. Backed by sessionStorage
// (per-tab, cleared on tab close) rather than localStorage, deliberately —
// a department picked last session would be a stale surprise days later.
//
// Explicit URL query params always win over this remembered default — the
// `departmentDefault`/`companyDefault` helpers only supply a fallback when
// the caller's own route-query read came back undefined, so a deep link or
// the Back button behaves exactly as before.
const STORAGE_KEY = 'internity:last-org-scope'

interface StoredScope {
  department_id?: string
  company_id?: string
}

function readScope(): StoredScope {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    return raw ? (JSON.parse(raw) as StoredScope) : {}
  } catch {
    return {}
  }
}

function writeScope(scope: StoredScope) {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(scope))
  } catch {
    // private-browsing / storage blocked — remembering the scope is a nicety, not required
  }
}

export function useLastOrgScope() {
  /** Falls back to the remembered department only when the route itself has none. */
  function departmentDefault(routeDepartmentId: number | undefined): number | undefined {
    if (routeDepartmentId !== undefined) return routeDepartmentId
    const stored = readScope().department_id
    return stored ? Number(stored) : undefined
  }

  /**
   * Falls back to the remembered company only when the route has none AND
   * the effective department (explicit or itself defaulted above) matches
   * the department that company was last picked under — otherwise it'd
   * resurrect a company that doesn't belong to the current department.
   */
  function companyDefault(routeCompanyId: number | undefined, effectiveDepartmentId: number | undefined): number | undefined {
    if (routeCompanyId !== undefined) return routeCompanyId
    const scope = readScope()
    if (scope.company_id && scope.department_id && effectiveDepartmentId !== undefined && String(effectiveDepartmentId) === scope.department_id) {
      return Number(scope.company_id)
    }
    return undefined
  }

  /** Call whenever the effective department/company changes so the next page picks it up. */
  function remember(departmentId: number | undefined, companyId: number | undefined) {
    writeScope({
      department_id: departmentId !== undefined ? String(departmentId) : undefined,
      company_id: companyId !== undefined ? String(companyId) : undefined,
    })
  }

  return { departmentDefault, companyDefault, remember }
}
