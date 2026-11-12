#!/usr/bin/env python3
"""Build a tiny native ELF/DWARF + defmt + metric dictionary and one parcel."""
import base64
import json
from pathlib import Path
import re
import struct
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
out = Path(sys.argv[1]) if len(sys.argv) > 1 else ROOT / 'symbolicator/fixtures'
out.mkdir(parents=True, exist_ok=True)
name = b'device.temperature'
metric_id = 0xcbf29ce484222325
for byte in name:
    metric_id = ((metric_id ^ byte) * 0x100000001b3) & ((1 << 64) - 1)
dictionary = b'SPKM' + bytes([1, 1]) + struct.pack('<HQ', len(name), metric_id) + name
source = r'''
__attribute__((used, section(".sparkly.metrics")))
const unsigned char dictionary[] = {BYTES};
__attribute__((used, section(".defmt"))) const char fixture_log = 0;
const char version __asm__("\"_defmt_version_ = 4\"") = 0;
const char encoding __asm__("\"_defmt_encoding_ = raw\"") = 0;
__attribute__((noinline)) int sampled_function(int x) { return x + 1; }
int main(void) { return sampled_function(4); }
'''.replace('BYTES', ','.join(map(str, dictionary)))
(out / 'firmware.c').write_text(source)
(out / 'metadata.x').write_text('SECTIONS { .defmt 0 (INFO) : { *(.defmt) } .sparkly.metrics 0 (INFO) : { *(.sparkly.metrics) } } INSERT AFTER .comment;\n')
elf = out / 'firmware.elf'
subprocess.run(['cc', '-g', '-O0', '-fno-pie', '-no-pie', '-Wl,--build-id=sha1',
                '-Wl,-T,' + str(out / 'metadata.x'), str(out / 'firmware.c'), '-o', str(elf)], check=True)
# A small real defmt table entry, indexed at address zero by metadata.x.
log_symbol = json.dumps({'package': 'fixture', 'disambiguator': '0', 'tag': 'defmt_info',
                         'data': 'temperature={=u8}', 'crate_name': 'fixture'}, separators=(',', ':'))
subprocess.run(['objcopy', '--redefine-sym', 'fixture_log=' + log_symbol, str(elf)], check=True)
build_id = re.search(r'Build ID: ([0-9a-f]+)', subprocess.check_output(['readelf', '-n', str(elf)], text=True)).group(1)
line = next(line for line in subprocess.check_output(['nm', str(elf)], text=True).splitlines() if line.endswith(' sampled_function'))
pc = int(line.split()[0], 16)
def b64(data):
    return base64.b64encode(data).decode('ascii')

boot = {'buildId': build_id, 'bootId': b64(b'0123456789abcdef'),
        'firmwareVersion': 'fixture', 'agentVersion': '0.1.0', 'resetReason': 'power_on'}
parcel = {
    'batchId': b64(b'0123456789abcdefghijklmn'),
    'initialBoot': boot,
    'events': [
        {'monotonicUs': '100', 'metric': {'metricId': str(metric_id), 'gauge': 23.5}},
        {'monotonicUs': '200', 'log': {'rawLog': {'level': 'LOG_LEVEL_INFO',
            'message': 'sensor ready', 'target': 'fixture'},
            'frames': [{'pc': str(pc)}], 'traceStatus': 'TRACE_STATUS_COMPLETE'}},
        {'monotonicUs': '300', 'log': {'defmtLog': b64(bytes([0, 0, 42]))}},
        {'monotonicUs': '400', 'crash': {'sp': '4096', 'stack': b64(bytes([1, 2])),
            'registers': {'a0': '42'}, 'reason': 'fixture fault', 'frames': [{'pc': str(pc)}]}},
        {'boot': boot},
    ],
}
(out / 'parcel.json').write_text(json.dumps(parcel, indent=2) + '\n')
print(f'Fixture written to {out}', file=sys.stderr)
