package aptfile

import (
	"fmt"
)

func ExampleFileCoord_LineAnnotated() {
	fmt.Println(
		FileCoord{
			Line:     "deb http://www.example.com",
			LineNum:  20,
			ColStart: 4,
			ColEnd:   26,
		}.LineAnnotated("argument with colon must be quoted"),
	)
	// Output:
	// 20 | deb http://www.example.com
	//          ^^^^^^^^^^^^^^^^^^^^^^ argument with colon must be quoted
}
