#!/usr/bin/env python3
"""Thin MLX-LM entry point that emits machine-readable memory telemetry."""
import json

import mlx.core as mx
from mlx_lm import lora


if __name__ == "__main__":
    lora.main()
    print("CNM_MLX_TELEMETRY=" + json.dumps({
        "active_memory": mx.metal.get_active_memory(),
        "cache_memory": mx.metal.get_cache_memory(),
        "peak_memory": mx.metal.get_peak_memory(),
        "device_info": mx.metal.device_info(),
    }, sort_keys=True), flush=True)
