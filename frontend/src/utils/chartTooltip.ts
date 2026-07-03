type ChartPoint = {
  x: number
  y: number
}

type ChartMetrics = {
  width: number
  height: number
}

const tooltipEdgeRoom = 112

export function edgeAwareTooltipStyle(point: ChartPoint, metrics: ChartMetrics) {
  const width = Math.max(metrics.width || 0, 1)
  const height = Math.max(metrics.height || 0, 1)
  const roomRight = width - point.x
  const roomLeft = point.x
  let transform = 'translate(-50%, calc(-100% - 10px))'

  if (roomRight < tooltipEdgeRoom) {
    transform = 'translate(calc(-100% - 12px), -100%)'
  } else if (roomLeft < tooltipEdgeRoom) {
    transform = 'translate(12px, -100%)'
  }

  return {
    left: `${(point.x / width) * 100}%`,
    top: `${(point.y / height) * 100}%`,
    transform
  }
}
