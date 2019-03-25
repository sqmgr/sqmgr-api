#!/usr/bin/env perl

=license

Copyright 2019 Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

=cut

use 5.016;
use warnings;
use File::Temp();

my $year    = (localtime)[5] + 1900;
my $license = <<"EOF";
Copyright $year Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
EOF

chomp( my @files = `find . \! -path '*/vendor/*' \! -path '*/.git/*' -type f \\( -name '*.go' -o -name '*.js' -o -name '*.css' -o -name '*.html' \\)` );

for my $file (@files) {
	open my $fh, '<', $file
		or die "could not read file $file: $!\n";
	my $content = do { local $/ = undef; <$fh> };
	close $fh;

	if ( $content =~ /\QApache License/x ) {
		next;
	}

	my $opening_comment = '/*';
	my $closing_comment = '*/';

	if ( $file =~ /\.html\z/ ) {
		$opening_comment = '{{/*';
		$closing_comment = '*/}}';
	}

	my $tmp = File::Temp->new;
	print $tmp "$opening_comment\n$license$closing_comment\n\n$content"
		or die "could not write to tempfile: $!\n";
	close $tmp
		or die "could not close tempfile: $!\n";

	say "adding license to $file...";
	rename $tmp->filename, $file
		or die "could not move ${ \$tmp->filename } to $file: $!\n";
}
