"""Generate Orix's code-native monochrome mark at launcher density sizes.
No reference raster is rescaled; every pixel is drawn from the same 32-unit grid.
"""
from pathlib import Path
import struct
import zlib
ROOT = Path(__file__).resolve().parents[1]

def mark(x,y):
    x-=4;y-=4
    ring=((6<=x<18 and 2<=y<22) or (2<=x<22 and 6<=y<18)) and not(8<=x<16 and 8<=y<16)
    return ring or (18<=x<24 and 18<=y<24)

def png(path,size,transparent=False):
    rows=[]
    for y in range(size):
        row=bytearray([0])
        for x in range(size):
            filled=mark(x*32/size,y*32/size)
            row.extend((34,34,34,255) if filled else (255,255,255,0 if transparent else 255))
        rows.append(row)
    def chunk(kind,data):return struct.pack('>I',len(data))+kind+data+struct.pack('>I',zlib.crc32(kind+data)&0xffffffff)
    path.parent.mkdir(parents=True,exist_ok=True)
    path.write_bytes(b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',size,size,8,6,0,0,0))+chunk(b'IDAT',zlib.compress(b''.join(rows)))+chunk(b'IEND',b''))
for density,size in [('mdpi',48),('hdpi',72),('xhdpi',96),('xxhdpi',144),('xxxhdpi',192)]:
    png(ROOT/f'android/app/src/main/res/mipmap-{density}/ic_launcher.png',size)
png(ROOT/'assets/Orix_Logo.png',1024)
png(ROOT/'assets/images/Orix_Logo.png',1024)
