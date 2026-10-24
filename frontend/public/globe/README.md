# 地球陆地掩码资源

`land-mask.png`：1024×512 等距柱状投影的陆地掩码，陆地 255、海洋 0，
供 `src/components/home/useOrbitScene.ts` 按陆地/海洋加权撒点。

## 来源与许可

- 底图数据：Natural Earth 1:110m `land`（`land-110m.json`，本目录同放），Natural Earth 数据属公有领域。
- 数据文件来自 `world-atlas@2`（https://github.com/topojson/world-atlas），该包同样基于 Natural Earth。
- 生成脚本：`build-land-mask.py`（本目录），仅依赖 Pillow。跨 180 度经线的环按经度跳变切段后逐段填充。

## 重新生成

```bash
python build-land-mask.py land-110m.json land-mask.png
```

脚本输出陆地占比，当前值约 0.31（地球陆地面积占比约 0.29）。
