"""Natural Earth 110m 陆地 TopoJSON → 等距柱状陆地掩码 PNG。

跨 180° 经线的环先按经度跳变切段，再逐段填充，避免出现横向拉丝。
用法: python build_land_mask.py <land-110m.json> <out.png>
"""
import json
import sys
from PIL import Image, ImageDraw

W, H = 1024, 512


def decode_arcs(topo):
    sx, sy = topo['transform']['scale']
    tx, ty = topo['transform']['translate']
    arcs = []
    for arc in topo['arcs']:
        x = y = 0
        pts = []
        for dx, dy in arc:
            x += dx
            y += dy
            pts.append((x * sx + tx, y * sy + ty))
        arcs.append(pts)
    return arcs


def ring_points(arcs, ring):
    pts = []
    for idx in ring:
        arc = arcs[idx] if idx >= 0 else list(reversed(arcs[-1 - idx]))
        pts.extend(arc if not pts else arc[1:])
    return pts


def split_on_seam(pts, limit=180.0):
    """按经度跳变把环切成若干段，避免跨经线时画出横贯整图的弦。"""
    segs = []
    cur = [pts[0]] if pts else []
    prev = pts[0][0] if pts else None
    for lon, lat in pts[1:]:
        if prev is not None and abs(lon - prev) > limit:
            segs.append(cur)
            cur = []
        cur.append((lon, lat))
        prev = lon
    if cur:
        segs.append(cur)
    return [s for s in segs if len(s) >= 3]


def to_pixels(pts):
    return [((lon + 180.0) / 360.0 * W, (90.0 - lat) / 180.0 * H) for lon, lat in pts]


def main(src, dst):
    topo = json.load(open(src, encoding='utf-8'))
    arcs = decode_arcs(topo)
    land = topo['objects']['land']

    img = Image.new('L', (W, H), 0)
    draw = ImageDraw.Draw(img)

    geoms = land['geometries'] if land['type'] == 'GeometryCollection' else [land]
    drawn = 0
    for geom in geoms:
        if geom['type'] == 'Polygon':
            polygons = [geom['arcs']]
        elif geom['type'] == 'MultiPolygon':
            polygons = geom['arcs']
        else:
            continue
        for polygon in polygons:
            for ring in polygon:
                for seg in split_on_seam(ring_points(arcs, ring)):
                    draw.polygon(to_pixels(seg), fill=255)
                    drawn += 1

    img.save(dst)
    land_ratio = sum(1 for p in img.getdata() if p > 127) / (W * H)
    print(f'segments={drawn} land_ratio={land_ratio:.4f} saved={dst}')


if __name__ == '__main__':
    main(sys.argv[1], sys.argv[2])
