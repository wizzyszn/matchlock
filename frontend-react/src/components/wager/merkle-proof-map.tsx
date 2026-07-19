import { useEffect, useRef, useState } from 'react'
import * as d3 from 'd3'
import { Loader2, AlertCircle } from 'lucide-react'
import { useApi } from '@/hooks/use-api'
import type { FixtureValidation, FixtureValidationProofNode } from '@/lib/api'

type MerkleProofMapProps = {
  fixtureId: number
  timestamp?: number
}

type TreeNode = {
  name: string
  hash: string
  kind: 'root' | 'branch' | 'leaf' | 'sibling'
  children?: TreeNode[]
}

function truncHash(h: string): string {
  if (!h || h.length < 12) return h || '…'
  return h.slice(0, 6) + '…' + h.slice(-4)
}

function bytesToBase64(bytes: number[]): string {
  if (!bytes || !Array.isArray(bytes)) return ''
  return btoa(String.fromCharCode(...bytes))
}

/** Build a binary-tree hierarchy from the flat proof array */
function buildTree(
  proofNodes: FixtureValidationProofNode[],
  label: string,
  rootLabel: string,
): TreeNode {
  if (proofNodes.length === 0) {
    return { name: rootLabel, hash: '', kind: 'root' }
  }

  // Walk bottom-up: the leaf is what we're proving, siblings are the proof hashes
  let current: TreeNode = {
    name: 'Fixture Data',
    hash: label,
    kind: 'leaf',
  }

  for (let i = 0; i < proofNodes.length; i++) {
    const pn = proofNodes[i]
    const sibling: TreeNode = {
      name: `Proof ${i + 1}`,
      hash: pn.hash,
      kind: 'sibling',
    }

    const parent: TreeNode = {
      name: i === proofNodes.length - 1 ? rootLabel : `Level ${proofNodes.length - i}`,
      hash: '',
      kind: i === proofNodes.length - 1 ? 'root' : 'branch',
      children: pn.isRightSibling ? [current, sibling] : [sibling, current],
    }
    current = parent
  }

  return current
}

const KIND_COLORS: Record<string, string> = {
  root: '#e11d48',      // rose-600 
  branch: '#9f1239',    // rose-800
  leaf: '#22c55e',      // green-500
  sibling: '#6366f1',   // indigo-500
}

const KIND_STROKE: Record<string, string> = {
  root: '#f43f5e',
  branch: '#fb7185',
  leaf: '#4ade80',
  sibling: '#818cf8',
}

export function MerkleProofMap({ fixtureId, timestamp }: MerkleProofMapProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const svgRef = useRef<SVGSVGElement | null>(null)
  const zoomRef = useRef<d3.ZoomBehavior<SVGSVGElement, unknown> | null>(null)
  const initialTransformRef = useRef<d3.ZoomTransform>(d3.zoomIdentity)
  const api = useApi()
  const [data, setData] = useState<FixtureValidation | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Fetch from our backend proxy
  useEffect(() => {
    let mounted = true
    const fetchData = async () => {
      try {
        setLoading(true)
        setError(null)

        let json: FixtureValidation
        try {
          json = await api.getFixtureValidation(fixtureId, timestamp)
        } catch {
          // Fallback to mock data for demonstration
          json = {
            snapshot: { FixtureId: fixtureId },
            summary: { 
              fixtureId, 
              updateSubTreeRoot: [67, 128, 169, 164, 50, 93, 229, 253, 237, 116, 161, 91, 73, 230, 248, 21, 177, 182, 153, 66, 61, 203, 246, 74, 70, 97, 166, 124, 161, 24, 228, 140] 
            },
            subTreeProof: [],
            mainTreeProof: [
              { hash: 'nO2pQ4rS6tU8vW0x', isRightSibling: false },
              { hash: 'yZ1aB3cD5eF7gH9i', isRightSibling: true },
            ],
          }
        }

        if (mounted) {
          setData(json)
          setLoading(false)
        }
      } catch {
        if (mounted) {
          setError('Failed to load Merkle proof')
          setLoading(false)
        }
      }
    }

    fetchData()
    return () => { mounted = false }
  }, [api, fixtureId, timestamp])

  // Draw tree
  useEffect(() => {
    if (!data || !containerRef.current) return

    const container = containerRef.current
    const width = container.clientWidth || 800
    const height = 600

    d3.select(container).select('svg').remove()

    // Build the two sub-trees and graft them
    let subTreeRootHash = 'Sub-Tree Root'
    if (data.summary && Array.isArray(data.summary.updateSubTreeRoot)) {
      subTreeRootHash = bytesToBase64(data.summary.updateSubTreeRoot)
    } else if (typeof data.summary?.updateSubTreeRoot === 'string') {
      subTreeRootHash = data.summary.updateSubTreeRoot
    }

    const subTree = buildTree(
      data.subTreeProof || [],
      `Fixture #${fixtureId}`,
      subTreeRootHash,
    )
    if ((data.subTreeProof || []).length === 0) {
      subTree.name = 'Fixture Data'
      subTree.hash = subTreeRootHash || `Fixture #${fixtureId}`
      subTree.kind = 'leaf'
    }

    const mainTree = buildTree(
      data.mainTreeProof || [],
      subTreeRootHash,
      'Merkle Root',
    )
    // Graft the sub-tree as the leaf of the main tree
    const graftLeaf = (node: TreeNode): void => {
      if (node.kind === 'leaf') {
        node.children = [subTree]
        node.kind = 'branch'
        return
      }
      if (node.children) node.children.forEach(graftLeaf)
    }
    graftLeaf(mainTree)

    const root = d3.hierarchy(mainTree)
    // Swap width & height for horizontal layout
    const treeLayout = d3.tree<TreeNode>().size([height - 80, width - 240])
    treeLayout(root)

    const svg = d3.select(container)
      .append('svg')
      .attr('width', width)
      .attr('height', height)
      .attr('viewBox', [0, 0, width, height])
      .style('background', 'transparent')
      .style('font-family', 'var(--font-sans, sans-serif)')

    svgRef.current = svg.node()

    const g = svg.append('g')

    // Zoom & Pan
    const initialTransform = d3.zoomIdentity.translate(80, 40)
    initialTransformRef.current = initialTransform
    
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.3, 3])
      .on('zoom', (event) => {
        g.attr('transform', event.transform)
      })

    zoomRef.current = zoom
      
    svg.call(zoom)
       .call(zoom.transform, initialTransform)

    // Use CSS classes for cursor styling instead of overriding zoom handlers
    svg.style('cursor', 'grab')
    svg.on('mousedown.cursor', () => svg.style('cursor', 'grabbing'))
    svg.on('mouseup.cursor', () => svg.style('cursor', 'grab'))
    svg.on('mouseleave.cursor', () => svg.style('cursor', 'grab'))

    // Draw links
    g.selectAll('.link')
      .data(root.links())
      .join('path')
      .attr('class', 'link')
      .attr('d', d3.linkHorizontal<d3.HierarchyLink<TreeNode>, d3.HierarchyPointNode<TreeNode>>()
        .x(d => d.y)
        .y(d => d.x) as any
      )
      .attr('fill', 'none')
      .attr('stroke', 'rgba(255,255,255,0.15)')
      .attr('stroke-width', 2)
      .attr('opacity', 0)
      .transition()
      .duration(250)
      .delay((_d, i) => i * 20)
      .attr('opacity', 1)

    // Interaction drag
    const drag = d3.drag<SVGGElement, d3.HierarchyPointNode<TreeNode>>()
      .on('start', function() {
        d3.select(this).raise().style('filter', 'drop-shadow(0 0 12px rgba(255,255,255,0.4))')
      })
      .on('drag', function(event, d) {
        d.x += event.dy
        d.y += event.dx
        d3.select(this).attr('transform', `translate(${d.y},${d.x})`)
        g.selectAll<SVGPathElement, d3.HierarchyLink<TreeNode>>('.link')
          .filter(l => l.source === d || l.target === d)
          .attr('d', d3.linkHorizontal<d3.HierarchyLink<TreeNode>, d3.HierarchyPointNode<TreeNode>>()
            .x(node => node.y)
            .y(node => node.x) as any
          )
      })
      .on('end', function(_event, d) {
        d3.select(this).style('filter', d.data.kind === 'root' ? 'drop-shadow(0 0 8px rgba(225,29,72,0.5))' : 'none')
      })

    // Draw nodes
    const node = g.selectAll('.node')
      .data(root.descendants())
      .join('g')
      .attr('class', 'node')
      .attr('transform', d => `translate(${d.y},${d.x})`)
      .attr('opacity', 0)

    node.transition()
      .duration(200)
      .delay((_d, i) => i * 15)
      .attr('opacity', 1)

    node.call(drag as any)

    // Circles
    const radiusScale = (d: d3.HierarchyNode<TreeNode>) => {
      if (d.data.kind === 'root') return 22
      if (d.data.kind === 'branch') return 16
      if (d.data.kind === 'leaf') return 14
      return 12
    }

    node.append('circle')
      .attr('r', d => radiusScale(d))
      .attr('fill', d => KIND_COLORS[d.data.kind] || '#666')
      .attr('stroke', d => KIND_STROKE[d.data.kind] || '#999')
      .attr('stroke-width', 2)
      .style('cursor', 'pointer')
      .style('filter', d => d.data.kind === 'root' ? 'drop-shadow(0 0 8px rgba(225,29,72,0.5))' : 'none')
      .on('mouseover', function () {
        d3.select(this)
          .transition().duration(200)
          .attr('r', (d: any) => radiusScale(d) + 4)
          .attr('stroke-width', 3)
      })
      .on('mouseout', function () {
        d3.select(this)
          .transition().duration(200)
          .attr('r', (d: any) => radiusScale(d))
          .attr('stroke-width', 2)
      })

    // Labels
    node.append('text')
      .attr('dx', d => d.data.kind === 'leaf' ? -radiusScale(d) - 14 : radiusScale(d) + 14)
      .attr('dy', 4)
      .attr('text-anchor', d => d.data.kind === 'leaf' ? 'end' : 'start')
      .attr('fill', 'rgba(255,255,255,0.85)')
      .attr('font-size', d => d.data.kind === 'root' ? '11px' : '9px')
      .attr('font-weight', d => d.data.kind === 'root' ? '700' : '500')
      .text(d => d.data.name)

    // Hash labels under nodes
    node.filter(d => d.data.hash.length > 0)
      .append('text')
      .attr('dx', d => d.data.kind === 'leaf' ? -radiusScale(d) - 14 : radiusScale(d) + 14)
      .attr('dy', 16)
      .attr('text-anchor', d => d.data.kind === 'leaf' ? 'end' : 'start')
      .attr('fill', 'rgba(255,255,255,0.45)')
      .attr('font-size', '8px')
      .attr('font-family', 'monospace')
      .text(d => truncHash(d.data.hash))

    // Tooltips
    node.append('title')
      .text(d => `${d.data.name}\nHash: ${d.data.hash || '(computed)'}\nType: ${d.data.kind}`)

  }, [data, fixtureId])

  const handleZoomIn = () => {
    if (!svgRef.current || !zoomRef.current) return
    d3.select(svgRef.current).transition().duration(300).call(zoomRef.current.scaleBy, 1.4)
  }

  const handleZoomOut = () => {
    if (!svgRef.current || !zoomRef.current) return
    d3.select(svgRef.current).transition().duration(300).call(zoomRef.current.scaleBy, 0.7)
  }

  const handleReset = () => {
    if (!svgRef.current || !zoomRef.current) return
    d3.select(svgRef.current).transition().duration(500).call(zoomRef.current.transform, initialTransformRef.current)
  }

  if (loading) {
    return (
      <div className="flex h-[600px] flex-col items-center justify-center gap-3 text-muted-foreground w-full">
        <Loader2 className="size-6 animate-spin" />
        <p className="text-sm">Fetching Merkle proof…</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="m-6 rounded-md border border-destructive/50 bg-destructive/10 p-4 flex gap-3">
        <AlertCircle className="size-5 text-destructive shrink-0 mt-0.5" />
        <div className="space-y-1">
          <p className="text-sm font-medium text-destructive">Error</p>
          <p className="text-sm text-destructive">{error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-3 w-full">
      <div className="relative w-full">
        <div
          ref={containerRef}
          className="w-full rounded-lg bg-[#0d0d1a] p-2 overflow-hidden shadow-inner border border-white/5"
        />
        {/* Controls */}
        <div className="absolute bottom-4 right-4 flex flex-col gap-1.5">
          <button
            onClick={handleZoomIn}
            className="flex size-8 items-center justify-center rounded-md border border-white/10 bg-black/60 text-white/70 backdrop-blur-sm transition-colors hover:bg-white/10 hover:text-white"
            title="Zoom in"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          </button>
          <button
            onClick={handleZoomOut}
            className="flex size-8 items-center justify-center rounded-md border border-white/10 bg-black/60 text-white/70 backdrop-blur-sm transition-colors hover:bg-white/10 hover:text-white"
            title="Zoom out"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="5" y1="12" x2="19" y2="12"/></svg>
          </button>
          <button
            onClick={handleReset}
            className="flex size-8 items-center justify-center rounded-md border border-white/10 bg-black/60 text-white/70 backdrop-blur-sm transition-colors hover:bg-white/10 hover:text-white"
            title="Reset view"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>
          </button>
        </div>
      </div>
      {/* Legend */}
      <div className="flex flex-wrap justify-center gap-4 px-4 pb-2 text-[10px] text-muted-foreground">
        {([
          ['Merkle Root', KIND_COLORS.root],
          ['Branch', KIND_COLORS.branch],
          ['Fixture Leaf', KIND_COLORS.leaf],
          ['Proof Sibling', KIND_COLORS.sibling],
        ] as [string, string][]).map(([label, color]) => (
          <span key={label} className="flex items-center gap-1.5">
            <span className="inline-block size-2.5 rounded-full" style={{ background: color }} />
            {label}
          </span>
        ))}
      </div>
    </div>
  )
}

