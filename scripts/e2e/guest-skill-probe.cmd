@echo off
REM Run the host's skill host once and capture what it says about loading skills.
REM The app itself only watches stdout, so skill load failures (which are logged
REM to stderr) are invisible in its logs - this makes them readable.
set SKILL_HOST="C:\Program Files\QuickDesk\quickdesk-skill-host.exe"
set SKILLS_DIR="C:\Program Files\QuickDesk\skills"
%SKILL_HOST% --skills-dir %SKILLS_DIR% < C:\QuickDeskTest\mcp-in.txt > C:\QuickDeskTest\sh-out.txt 2> C:\QuickDeskTest\sh-err.txt
