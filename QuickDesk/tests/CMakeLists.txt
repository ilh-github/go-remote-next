add_executable(quickdesk_certificate_pin_test
    certificatepin_test.cpp
    ../src/core/certificatepin.cpp
)

target_include_directories(quickdesk_certificate_pin_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

add_test(
    NAME quickdesk_certificate_pin_test
    COMMAND quickdesk_certificate_pin_test
)
