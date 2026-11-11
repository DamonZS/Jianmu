//go:build darwin && cgo

package gui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#include <stdlib.h>

// The Dock tile pulses as the header logo does: the icon the system draws
// for jianmu (its glass tile and glossy mark, light) has the mark cut out
// of it, and each frame draws that mark over the tile swelled or settled
// about its middle. Drawn at mpDockPx; the tile's view scales it to the
// Dock's size.
static const int mpDockPx = 512;
// where the system puts the mark on the canvas: centred, 0.64 of it across
// (a 0.8 tile, the glyph 0.8 of that), as build/icon/gen.go's app() and
// AppIcon.icon lay it
static const CGFloat mpDockBird = 0.64;

static CGImageRef mpDockClean; // the tile, the mark taken out
static CGImageRef mpDockMark;  // the mark, with alpha
static CGPathRef mpDockMarkPath; // the mark's outline, kept for the second, wider cut
static NSImageView *mpDockView;

static CGPathRef mpDockPath(const char *ops, int nops, const double *pts) {
	CGMutablePathRef p = CGPathCreateMutable();
	for (int i = 0, j = 0; i < nops; i++) {
		switch (ops[i]) {
		case 'M': CGPathMoveToPoint(p, NULL, pts[j], pts[j+1]); j += 2; break;
		case 'L': CGPathAddLineToPoint(p, NULL, pts[j], pts[j+1]); j += 2; break;
		case 'C': CGPathAddCurveToPoint(p, NULL, pts[j], pts[j+1], pts[j+2], pts[j+3], pts[j+4], pts[j+5]); j += 6; break;
		case 'Z': CGPathCloseSubpath(p); break;
		}
	}
	return p;
}

static CGContextRef mpDockContext(void) {
	CGColorSpaceRef cs = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
	CGContextRef c = CGBitmapContextCreate(NULL, mpDockPx, mpDockPx, 8, 0, cs, (CGBitmapInfo)kCGImageAlphaPremultipliedLast);
	CGColorSpaceRelease(cs);
	return c;
}

// mpDockUnits puts the viewBox's 44 units, y down, where the system draws
// the mark.
static void mpDockUnits(CGContextRef c) {
	CGFloat k = mpDockPx * mpDockBird / 44, o = mpDockPx / 2.0 - 22 * k;
	CGContextTranslateCTM(c, o, mpDockPx - o);
	CGContextScaleCTM(c, k, -k);
}

// The mark's ink reaches a little past its path: its antialiased and glossy
// edge. Each cut takes this much more, in pixels, so no edge is left on
// the tile.
static const CGFloat mpDockEdge = 4;

// mpDockFill is a grey mask, in the canvas's pixels, of path filled
// even-odd and stroked out by edge pixels.
static CGImageRef mpDockFill(CGPathRef path, CGFloat edge) {
	CGColorSpaceRef g = CGColorSpaceCreateDeviceGray();
	CGContextRef c = CGBitmapContextCreate(NULL, mpDockPx, mpDockPx, 8, 0, g, (CGBitmapInfo)kCGImageAlphaNone);
	CGColorSpaceRelease(g);
	CGContextSetGrayFillColor(c, 0, 1);
	CGContextFillRect(c, CGRectMake(0, 0, mpDockPx, mpDockPx));
	mpDockUnits(c);
	CGContextSetGrayFillColor(c, 1, 1);
	CGContextSetGrayStrokeColor(c, 1, 1);
	CGContextAddPath(c, path);
	CGContextEOFillPath(c);
	CGContextSetLineWidth(c, 2 * edge * 44 / (mpDockPx * mpDockBird));
	CGContextSetLineJoin(c, kCGLineJoinRound);
	CGContextAddPath(c, path);
	CGContextStrokePath(c);
	CGImageRef m = CGBitmapContextCreateImage(c);
	CGContextRelease(c);
	return m;
}

// mpDockReady cuts the icon up once: the mark, and the tile with the mark
// filled with the tile's own colour, read either side of it on each row and
// blended across. 0 when the icon can't be had.
static int mpDockReady(const char *app, const char *ops, int nops, const double *pts) {
	if (mpDockClean) return 1;
	@autoreleasepool {
	NSString *path = app ? [NSString stringWithUTF8String:app] : [[NSBundle mainBundle] bundlePath];
	NSImage *icon = [[NSWorkspace sharedWorkspace] iconForFile:path];
	if (!icon) return 0;
	CGContextRef c = mpDockContext();
	if (!c) return 0;
	NSGraphicsContext *ns = [NSGraphicsContext graphicsContextWithCGContext:c flipped:NO];
	[NSGraphicsContext saveGraphicsState];
	[NSGraphicsContext setCurrentContext:ns];
	[[NSAppearance appearanceNamed:NSAppearanceNameAqua] performAsCurrentDrawingAppearance:^{
		[icon drawInRect:NSMakeRect(0, 0, mpDockPx, mpDockPx) fromRect:NSZeroRect operation:NSCompositingOperationCopy fraction:1];
	}];
	[NSGraphicsContext restoreGraphicsState];
	CGImageRef full = CGBitmapContextCreateImage(c);
	mpDockMarkPath = mpDockPath(ops, nops, pts);

	CGImageRef m = mpDockFill(mpDockMarkPath, mpDockEdge);
	CGContextRef p = mpDockContext();
	CGContextClipToMask(p, CGRectMake(0, 0, mpDockPx, mpDockPx), m);
	CGContextDrawImage(p, CGRectMake(0, 0, mpDockPx, mpDockPx), full);
	mpDockMark = CGBitmapContextCreateImage(p);
	CGContextRelease(p);
	CGImageRelease(m);

	// the mark out: a wider cut than the mark takes, so its swelled edges
	// leave the tile's colour, not its old edge
	m = mpDockFill(mpDockMarkPath, mpDockEdge + 1);
	CFDataRef md = CGDataProviderCopyData(CGImageGetDataProvider(m));
	size_t mrow = CGImageGetBytesPerRow(m);
	CGImageRelease(m);
	const uint8_t *mask = CFDataGetBytePtr(md);
	uint8_t *px = CGBitmapContextGetData(c);
	size_t row = CGBitmapContextGetBytesPerRow(c);
	// either side of the mark, inside the tile and its rim: the mark spans
	// 0.18 to 0.82 of the canvas, the tile 0.1 to 0.9
	int l0 = mpDockPx * 0.12, l1 = mpDockPx * 0.16, r0 = mpDockPx * 0.84, r1 = mpDockPx * 0.87;
	CGFloat lx = (l0 + l1) / 2.0, rx = (r0 + r1) / 2.0;
	for (int y = 0; y < mpDockPx; y++) {
		uint8_t *r = px + y * row;
		// both bitmaps are top row first
		const uint8_t *mr = mask + y * mrow;
		double L[4] = {0}, R[4] = {0};
		for (int x = l0; x < l1; x++) for (int k = 0; k < 4; k++) L[k] += r[4*x+k];
		for (int x = r0; x < r1; x++) for (int k = 0; k < 4; k++) R[k] += r[4*x+k];
		for (int k = 0; k < 4; k++) { L[k] /= (l1 - l0); R[k] /= (r1 - r0); }
		for (int x = 0; x < mpDockPx; x++) {
			if (!mr[x]) continue;
			double a = mr[x] / 255.0, f = fmin(1, fmax(0, (x - lx) / (rx - lx)));
			for (int k = 0; k < 4; k++) {
				double bg = L[k] * (1 - f) + R[k] * f;
				r[4*x+k] = (uint8_t)lround(r[4*x+k] * (1 - a) + bg * a);
			}
		}
	}
	CFRelease(md);
	mpDockClean = CGBitmapContextCreateImage(c);
	CGContextRelease(c);
	CGImageRelease(full);
	return mpDockClean != NULL;
	}
}

// mpDockFrameImage draws a frame: the tile, and the mark over it scaled
// about the canvas's middle, as CSS's scale() about the viewBox's (22, 22)
// does.
static CGImageRef mpDockFrameImage(double size) {
	CGContextRef c = mpDockContext();
	if (!c) return NULL;
	CGContextSetInterpolationQuality(c, kCGInterpolationHigh);
	CGRect all = CGRectMake(0, 0, mpDockPx, mpDockPx);
	CGContextDrawImage(c, all, mpDockClean);
	CGFloat mid = mpDockPx / 2.0;
	CGContextTranslateCTM(c, mid, mid);
	CGContextScaleCTM(c, size, size);
	CGContextTranslateCTM(c, -mid, -mid);
	CGContextDrawImage(c, all, mpDockMark);
	CGImageRef im = CGBitmapContextCreateImage(c);
	CGContextRelease(c);
	return im;
}

// mpDockPNG writes a frame to a file, to look at.
static int mpDockPNG(const char *file, double size) {
	CGImageRef im = mpDockFrameImage(size);
	if (!im) return 0;
	@autoreleasepool {
		NSBitmapImageRep *rep = [[[NSBitmapImageRep alloc] initWithCGImage:im] autorelease];
		CGImageRelease(im);
		NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
		return [png writeToFile:[NSString stringWithUTF8String:file] atomically:YES];
	}
}

// mpDockFrameNS is a frame as an NSImage, retained. It is made on a Go
// thread, which has no autorelease pool: without one of its own, what
// AppKit autoreleases in making it (its NSCGImageSnapshotRep, holding the
// frame's 1 MB of pixels) was never let go, 29 frames a pulse, for as long
// as jianmu ran (#4162).
static NSImage *mpDockFrameNS(CGImageRef im) {
	@autoreleasepool {
		return [[NSImage alloc] initWithCGImage:im size:NSMakeSize(mpDockPx, mpDockPx)];
	}
}

// mpDockFrameFreed makes a frame's NSImage as mpDockShow does, on the
// calling thread, lets it go, and says whether its pixels went with it:
// 1 when nothing of it is left, for tests.
static int mpDockFrameFreed(void) {
	CGContextRef c = mpDockContext();
	if (!c) return -1;
	CGImageRef im = CGBitmapContextCreateImage(c);
	CGContextRelease(c);
	if (!im) return -1;
	NSImage *ns = mpDockFrameNS(im);
	CGImageRelease(im);
	// a weak reference to its pixels' rep; each look at it in a pool of
	// its own, since objc_loadWeak autoreleases what it gives
	id rep = nil;
	int seen, freed;
	@autoreleasepool {
		objc_storeWeak(&rep, ns.representations.firstObject);
		seen = objc_loadWeak(&rep) != nil;
	}
	[ns release];
	@autoreleasepool {
		freed = objc_loadWeak(&rep) == nil;
		objc_storeWeak(&rep, nil);
	}
	if (!seen) return -1; // nothing to watch
	return freed;
}

static void mpDockShow(double size) {
	CGImageRef im = mpDockFrameImage(size);
	if (!im) return;
	NSImage *ns = mpDockFrameNS(im);
	CGImageRelease(im);
	dispatch_async(dispatch_get_main_queue(), ^{
		NSDockTile *tile = NSApp.dockTile;
		if (!mpDockView) {
			mpDockView = [[NSImageView alloc] initWithFrame:NSMakeRect(0, 0, tile.size.width, tile.size.height)];
			mpDockView.imageScaling = NSImageScaleProportionallyUpOrDown;
		}
		mpDockView.image = ns;
		[ns release];
		if (tile.contentView != mpDockView) tile.contentView = mpDockView;
		[tile display];
	});
}

// mpDockStill gives the tile back to the system's icon.
static void mpDockStill(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSDockTile *tile = NSApp.dockTile;
		tile.contentView = nil;
		mpDockView.image = nil; // the last frame's pixels, until the next pulse
		[tile display];
	});
}

// mpDockPlain is 1 when the Dock draws icons as they are, light: the
// cut is of that drawing, and a dark, clear or tinted Dock would flash it.
static int mpDockPlain(void) {
	CFPropertyListRef v = CFPreferencesCopyAppValue(CFSTR("AppleIconAppearanceTheme"), kCFPreferencesAnyApplication);
	if (!v) return 1;
	int plain = 0;
	if (CFGetTypeID(v) == CFStringGetTypeID()) {
		NSString *s = (__bridge NSString *)v;
		plain = [s isEqualToString:@"RegularLight"];
		if ([s isEqualToString:@"RegularAutomatic"]) {
			NSAppearanceName n = [NSApp.effectiveAppearance bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]];
			plain = [n isEqualToString:NSAppearanceNameAqua];
		}
	}
	CFRelease(v);
	return plain;
}
*/
import "C"

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	dockOnce    sync.Once
	dockGood    bool
	dockPulsing atomic.Bool
)

// dockReady cuts the icon of app (jianmu's own bundle when "") up the first
// time the tile pulses, by the header logo's paths; false when it can't, or
// the Dock shows icons another way.
func dockReady(app string) bool {
	if C.mpDockPlain() == 0 {
		return false
	}
	dockOnce.Do(func() {
		page, err := assets.ReadFile("assets/index.html")
		if err != nil {
			return
		}
		rings, ok := logoPaths(string(page))
		if !ok {
			return
		}
		mark := rings[0]
		// The rings are separate subpaths, and C fills the whole path
		// even-odd, so the bands come out as they do in the page.
		for _, r := range rings[1:] {
			mark.ops = append(mark.ops, r.ops...)
			mark.pts = append(mark.pts, r.pts...)
		}
		var capp *C.char
		if app != "" {
			capp = C.CString(app)
			defer C.free(unsafe.Pointer(capp))
		}
		ops, pts := C.CBytes(mark.ops), C.CBytes(floats(mark.pts))
		defer C.free(ops)
		defer C.free(pts)
		dockGood = C.mpDockReady(capp, (*C.char)(ops), C.int(len(mark.ops)), (*C.double)(pts)) != 0
	})
	return dockGood
}

// floats is a []float64 as C doubles' bytes.
func floats(f []float64) []byte {
	if len(f) == 0 {
		return []byte{0}
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&f[0])), len(f)*8)
}

// dockFlap plays the logo's pulse on the Dock tile, a frame every
// flapEvery, and gives the tile back to the system's icon after. A click
// while it plays doesn't start another.
func dockFlap() {
	if !dockPulsing.CompareAndSwap(false, true) {
		return
	}
	defer dockPulsing.Store(false)
	if !dockReady("") {
		return
	}
	n := int(flapSecs/flapEvery + 0.5)
	start := time.Now()
	for f := 1; f < n; f++ {
		C.mpDockShow(C.double(markAt(float64(f) / float64(n))))
		// on the pulse's clock, not the frames': a slow frame is skipped past
		time.Sleep(time.Until(start.Add(time.Duration(float64(f+1) * flapEvery * float64(time.Second)))))
	}
	C.mpDockStill()
}

// dockFramePNG writes the tile at t, 0..1 of the pulse, cut from app's icon.
func dockFramePNG(app, file string, t float64) bool {
	if !dockReady(app) {
		return false
	}
	cf := C.CString(file)
	defer C.free(unsafe.Pointer(cf))
	return C.mpDockPNG(cf, C.double(markAt(t))) != 0
}

// dockFrameFreed makes a Dock frame's image on this goroutine's thread, as
// dockFlap does, lets it go, and says whether its pixels went with it; ok
// is false when there was nothing to watch.
func dockFrameFreed() (freed, ok bool) {
	r := C.mpDockFrameFreed()
	return r == 1, r >= 0
}
