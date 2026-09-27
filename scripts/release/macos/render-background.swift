// Regenerate the deterministic 1x/2x Finder background using macOS system fonts.
// Usage: swift render-background.swift <repo-root> <output-directory>
import AppKit

let root = URL(fileURLWithPath: CommandLine.arguments[1])
let output = URL(fileURLWithPath: CommandLine.arguments[2])
try FileManager.default.createDirectory(at: output, withIntermediateDirectories: true)
let width = 680.0, height = 440.0
func color(_ hex: UInt32) -> NSColor {
    NSColor(srgbRed: CGFloat((hex >> 16) & 255) / 255,
            green: CGFloat((hex >> 8) & 255) / 255,
            blue: CGFloat(hex & 255) / 255, alpha: 1)
}
let ink = color(0x302A31), secondary = color(0x716771), pink = color(0xE54D84)
let wordmark = NSImage(contentsOf: root.appendingPathComponent("icon/curated-wordmark.png"))!
for scale in [1, 2] {
    let bitmap = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: Int(width) * scale,
        pixelsHigh: Int(height) * scale, bitsPerSample: 8, samplesPerPixel: 4,
        hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
    bitmap.size = NSSize(width: width, height: height)
    let context = NSGraphicsContext(bitmapImageRep: bitmap)!
    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = context
    color(0xFCF9FB).setFill()
    NSBezierPath(rect: NSRect(x: 0, y: 0, width: width, height: height)).fill()
    // Coordinates below use a top-left origin; text remains upright.
    func rect(_ x: Double, _ y: Double, _ w: Double, _ h: Double) -> NSRect {
        NSRect(x: x, y: height-y-h, width: w, height: h)
    }
    func text(_ value: String, y: Double, size: Double, weight: NSFont.Weight, tint: NSColor) {
        let paragraph = NSMutableParagraphStyle()
        paragraph.alignment = .center
        (value as NSString).draw(in: rect(32, y, width-64, size*1.6), withAttributes: [
            .font: NSFont.systemFont(ofSize: size, weight: weight),
            .foregroundColor: tint, .paragraphStyle: paragraph
        ])
    }
    wordmark.draw(in: rect(230, 20, 220, 65.3))
    text("拖入应用程序，即可安装", y: 100, size: 23, weight: .medium, tint: ink)
    text("Drag the app into Applications to install.", y: 136, size: 14, weight: .regular, tint: secondary)
    // Quiet landing areas behind the real, draggable Finder icons.
    for x in [190.0, 490.0] {
        color(0xF5EDF2).setFill()
        NSBezierPath(ovalIn: rect(x-72, 172, 144, 144)).fill()
    }
    let arrow = NSBezierPath()
    arrow.move(to: NSPoint(x: 310, y: height-244))
    arrow.line(to: NSPoint(x: 369, y: height-244))
    arrow.move(to: NSPoint(x: 357, y: height-232))
    arrow.line(to: NSPoint(x: 369, y: height-244))
    arrow.line(to: NSPoint(x: 357, y: height-256))
    arrow.lineWidth = 2.5
    arrow.lineCapStyle = .round
    arrow.lineJoinStyle = .round
    pink.setStroke(); arrow.stroke()
    color(0xE9E0E6).setFill()
    NSBezierPath(rect: rect(64, 352, 552, 1)).fill()
    text("安装后，从「应用程序」打开 Curated Desktop。", y: 374, size: 13, weight: .regular, tint: ink)
    text("Once installed, open Curated Desktop from Applications.", y: 397, size: 12, weight: .regular, tint: secondary)
    NSGraphicsContext.restoreGraphicsState()
    let name = scale == 1 ? "background.png" : "background@2x.png"
    try bitmap.representation(using: .png, properties: [:])!.write(to: output.appendingPathComponent(name))
}
