import type { Department, Employee } from '../api'

interface Props {
  employees: Employee[]
  departments: Department[]
  meId: number
}

interface Node extends Employee {
  children: Node[]
}

function buildForest(employees: Employee[]): Node[] {
  const byId = new Map<number, Node>(employees.map((e) => [e.id, { ...e, children: [] }]))
  const roots: Node[] = []
  for (const node of byId.values()) {
    if (node.manager_id !== null && byId.has(node.manager_id)) {
      byId.get(node.manager_id)!.children.push(node)
    } else {
      roots.push(node)
    }
  }
  for (const node of byId.values()) {
    node.children.sort((a, b) => a.full_name.localeCompare(b.full_name))
  }
  roots.sort((a, b) => a.full_name.localeCompare(b.full_name))
  return roots
}

const roleLabel: Record<string, string> = {
  employee: 'Employee',
  manager: 'Manager',
  hr_admin: 'HR Admin',
}

function designationOf(node: Employee) {
  return node.title ?? roleLabel[node.role] ?? node.role
}

function TreeNode({ node, meId }: { node: Node; meId: number }) {
  const isMe = node.id === meId
  const designation = designationOf(node)

  return (
    <div className="tree-node-wrap">
      <div className={isMe ? 'tree-node tree-node-me' : 'tree-node'}>
        <span className="tree-node-name">{node.full_name}</span>
        <span className="tree-node-meta">{designation}</span>
      </div>
      {node.children.length > 0 && (
        <div className="tree-children">
          {node.children.map((child) => (
            <TreeNode key={child.id} node={child} meId={meId} />
          ))}
        </div>
      )}
    </div>
  )
}

// HR Admin is a permission level, not a rung on the reporting ladder, so it never appears
// inside the manager/report tree, even for someone whose own account holds it. Below that,
// each department gets its own clearly headed, separate tree: without that grouping, one
// manager's reports sit directly above the next manager's own card with no visual break,
// which reads as if the second manager reported to the first.
export default function OrgTree({ employees, departments, meId }: Props) {
  const hrAdmins = employees.filter((e) => e.role === 'hr_admin')
  const roots = buildForest(employees.filter((e) => e.role !== 'hr_admin'))

  const groups = departments
    .map((dept) => ({ dept, roots: roots.filter((r) => r.department_id === dept.id) }))
    .filter((g) => g.roots.length > 0)
  const unassigned = roots.filter((r) => !departments.some((d) => d.id === r.department_id))

  return (
    <div>
      {hrAdmins.length > 0 && (
        <div className="org-admin-row">
          {hrAdmins.map((admin) => (
            <div key={admin.id} className={admin.id === meId ? 'tree-node tree-node-admin tree-node-me' : 'tree-node tree-node-admin'}>
              <span className="tree-node-name">{admin.full_name}</span>
              <span className="tree-node-meta">{designationOf(admin)}, company-wide, outside the reporting line</span>
            </div>
          ))}
        </div>
      )}
      {groups.map(({ dept, roots: deptRoots }) => (
        <div key={dept.id} className="org-department-group">
          <h4 className="org-department-heading">{dept.name}</h4>
          <div className="org-tree">
            {deptRoots.map((root) => (
              <TreeNode key={root.id} node={root} meId={meId} />
            ))}
          </div>
        </div>
      ))}
      {unassigned.length > 0 && (
        <div className="org-department-group">
          <h4 className="org-department-heading">No department</h4>
          <div className="org-tree">
            {unassigned.map((root) => (
              <TreeNode key={root.id} node={root} meId={meId} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
