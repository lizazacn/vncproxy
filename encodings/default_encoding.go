package encodings

import "github.com/lizazacn/vncproxy/rfb"

var (
	DefaultEncodings = []rfb.IEncoding{
		&ZRLEEncoding{},
		&TightEncoding{},
		&HexTileEncoding{},
		&TightPngEncoding{},
		&RREEncoding{},
		&ZLibEncoding{},
		&CopyRectEncoding{},
		&CoRREEncoding{},
		&RawEncoding{},
		&CursorPseudoEncoding{},
		&DesktopNamePseudoEncoding{},
		&DesktopSizePseudoEncoding{},
		&CursorPosPseudoEncoding{},
		&ExtendedDesktopSizePseudo{},
		&CursorWithAlphaPseudoEncoding{},
		&LedStatePseudo{},
		&LastRectPseudo{},
		&FencePseudo{},
		&XCursorPseudoEncoding{},
	}
)
