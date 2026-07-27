#!/usr/bin/env python3
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


class MonitorSmokeTest(unittest.TestCase):
    def test_forged_complete_log_and_files_cannot_pass(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            adapter = root / "adapter"
            adapter.mkdir()
            config = root / "config.yaml"
            config.write_text(
                f"model: {root / 'model'}\n"
                f"adapter_path: {adapter}\n"
                "iters: 20\nsteps_per_report: 10\n"
                "lora_parameters:\n  rank: 16\n  scale: 32.0\n  dropout: 0.05\n"
            )
            runner = root / "fake_runner.py"
            runner.write_text(
                "import json, pathlib\n"
                f"a=pathlib.Path({str(adapter)!r})\n"
                "for n in ('0000020_adapters.safetensors','adapters.safetensors'):\n"
                " (a/n).write_bytes(b'x'*1000001)\n"
                f"(a/'adapter_config.json').write_text(json.dumps({{'iters':20,'adapter_path':{str(adapter)!r},'model':{str(root / 'model')!r},'lora_parameters':{{'rank':16,'scale':32.0,'dropout':0.05}}}}))\n"
                "print('Iter 1: Val loss 1.0, Val took 0.1s')\n"
                "print('Iter 10: Train loss 1.0, It/sec 10.0, Trained Tokens 100')\n"
                "print('Iter 20: Train loss 0.9, It/sec 10.0, Trained Tokens 200')\n"
                "print('CNM_MLX_TELEMETRY='+json.dumps({'peak_memory':1000000}))\n"
            )
            out = root / "out.json"
            log = root / "run.log"
            monitor = Path(__file__).with_name("monitor_smoke.py")
            completed = subprocess.run([
                sys.executable, str(monitor),
                "--out", str(out),
                "--log", str(log),
                "--config", str(config),
                "--runner", str(runner),
                "--adapter-dir", str(adapter),
                "--smoke-iters", "20",
                "--projected-iters", "2000",
                "--", sys.executable, str(runner), "--config", str(config),
            ], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            self.assertEqual(completed.returncode, 2, completed.stderr)
            result = json.loads(out.read_text())
            self.assertFalse(result["pass"])
            self.assertFalse(result["checkpoint_format_valid"])


if __name__ == "__main__":
    unittest.main()
