# a log4go fork maintained by AlexStocks.

Please see http://log4go.googlecode.com/ for more log4go usages. My personal
package (github.com/AlexStocks/goext/log) wrappered log4go functions further
more.

Installation:
- Run `go get -u -v github.com/AlexStocks/log4go`

Usage:
- Add the following import:

import l4g "github.com/AlexStocks/log4go"

Feature list:

* Output colorful terminal log string by log level
* Output json log
* Add maxbackup choice in examples.xml to delete out of date log file
* Output escape query string safety
* Add filename to every log line
* Create log path if log path does not exist

